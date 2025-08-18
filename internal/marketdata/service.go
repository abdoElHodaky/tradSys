package marketdata

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	pb "github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the MarketDataService gRPC interface
type Service struct {
	pb.UnimplementedMarketDataServiceServer
	logger     *zap.Logger
	quotes     map[string]*pb.Quote
	mu         sync.RWMutex
	repository *repositories.MarketDataRepository
	subscribers map[string]map[pb.MarketDataService_SubscribeQuotesServer]bool
	subMu       sync.RWMutex
}

// NewService creates a new market data service
func NewService(logger *zap.Logger, repository *repositories.MarketDataRepository) *Service {
	service := &Service{
		logger:      logger,
		quotes:      make(map[string]*pb.Quote),
		repository:  repository,
		subscribers: make(map[string]map[pb.MarketDataService_SubscribeQuotesServer]bool),
	}
	
	// Start background tasks
	go service.persistQuotesPeriodically()
	
	return service
}

// GetQuote returns the latest quote for a symbol
func (s *Service) GetQuote(ctx context.Context, req *pb.MarketDataRequest) (*pb.Quote, error) {
	// Try to get from memory first for lowest latency
	s.mu.RLock()
	key := req.Symbol + ":" + req.Exchange
	quote, ok := s.quotes[key]
	s.mu.RUnlock()
	
	if ok {
		return quote, nil
	}
	
	// If not in memory, try to get from database
	dbQuote, err := s.repository.GetLatestQuote(ctx, req.Symbol, req.Exchange)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	
	if dbQuote == nil {
		return nil, status.Errorf(codes.NotFound, "quote not found for %s:%s", req.Symbol, req.Exchange)
	}
	
	// Convert database model to protobuf
	protoQuote := &pb.Quote{
		Symbol:    dbQuote.Symbol,
		Bid:       dbQuote.Bid,
		Ask:       dbQuote.Ask,
		BidSize:   dbQuote.BidSize,
		AskSize:   dbQuote.AskSize,
		Timestamp: dbQuote.Timestamp.UnixNano(),
		Exchange:  dbQuote.Exchange,
	}
	
	// Cache in memory for future requests
	s.mu.Lock()
	s.quotes[key] = protoQuote
	s.mu.Unlock()
	
	return protoQuote, nil
}

// SubscribeQuotes streams quotes for a requested symbol
func (s *Service) SubscribeQuotes(req *pb.MarketDataRequest, stream pb.MarketDataService_SubscribeQuotesServer) error {
	key := req.Symbol + ":" + req.Exchange
	
	// Register subscriber
	s.subMu.Lock()
	if _, exists := s.subscribers[key]; !exists {
		s.subscribers[key] = make(map[pb.MarketDataService_SubscribeQuotesServer]bool)
	}
	s.subscribers[key][stream] = true
	s.subMu.Unlock()
	
	s.logger.Info("Client subscribed to quotes",
		zap.String("symbol", req.Symbol),
		zap.String("exchange", req.Exchange))
	
	// Send initial quote if available
	s.mu.RLock()
	quote, ok := s.quotes[key]
	s.mu.RUnlock()
	
	if ok {
		if err := stream.Send(quote); err != nil {
			s.logger.Error("Failed to send initial quote",
				zap.Error(err),
				zap.String("symbol", req.Symbol))
		}
	}
	
	// Keep the stream open until client disconnects
	<-stream.Context().Done()
	
	// Unregister subscriber
	s.subMu.Lock()
	if subs, exists := s.subscribers[key]; exists {
		delete(subs, stream)
		if len(subs) == 0 {
			delete(s.subscribers, key)
		}
	}
	s.subMu.Unlock()
	
	s.logger.Info("Client unsubscribed from quotes",
		zap.String("symbol", req.Symbol),
		zap.String("exchange", req.Exchange))
	
	return nil
}

// GetHistoricalData retrieves historical market data
func (s *Service) GetHistoricalData(ctx context.Context, req *pb.HistoricalDataRequest) (*pb.QuoteList, error) {
	startTime := time.Unix(0, req.StartTime)
	endTime := time.Unix(0, req.EndTime)
	
	// Get historical data from database
	quotes, err := s.repository.GetQuoteHistory(ctx, req.Symbol, req.Exchange, startTime, endTime, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	
	// Convert to protobuf
	protoQuotes := make([]*pb.Quote, 0, len(quotes))
	for _, q := range quotes {
		protoQuotes = append(protoQuotes, &pb.Quote{
			Symbol:    q.Symbol,
			Bid:       q.Bid,
			Ask:       q.Ask,
			BidSize:   q.BidSize,
			AskSize:   q.AskSize,
			Timestamp: q.Timestamp.UnixNano(),
			Exchange:  q.Exchange,
		})
	}
	
	return &pb.QuoteList{Quotes: protoQuotes}, nil
}

// UpdateQuote updates the quote for a symbol and notifies subscribers
func (s *Service) UpdateQuote(quote *pb.Quote) {
	key := quote.Symbol + ":" + quote.Exchange
	
	// Update in-memory cache
	s.mu.Lock()
	s.quotes[key] = quote
	s.mu.Unlock()
	
	// Notify subscribers
	s.subMu.RLock()
	if subs, exists := s.subscribers[key]; exists {
		for stream := range subs {
			if err := stream.Send(quote); err != nil {
				s.logger.Error("Failed to send quote update",
					zap.Error(err),
					zap.String("symbol", quote.Symbol))
				// We'll clean up dead streams in the next subscription request
			}
		}
	}
	s.subMu.RUnlock()
}

// persistQuotesPeriodically periodically persists quotes to the database
func (s *Service) persistQuotesPeriodically() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		s.mu.RLock()
		quotes := make([]*pb.Quote, 0, len(s.quotes))
		for _, quote := range s.quotes {
			quotes = append(quotes, quote)
		}
		s.mu.RUnlock()
		
		for _, quote := range quotes {
			dbQuote := &models.Quote{
				Symbol:    quote.Symbol,
				Exchange:  quote.Exchange,
				Bid:       quote.Bid,
				Ask:       quote.Ask,
				BidSize:   quote.BidSize,
				AskSize:   quote.AskSize,
				Timestamp: time.Unix(0, quote.Timestamp),
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := s.repository.SaveQuote(ctx, dbQuote); err != nil {
				s.logger.Error("Failed to persist quote",
					zap.Error(err),
					zap.String("symbol", quote.Symbol))
			}
			cancel()
		}
	}
}

