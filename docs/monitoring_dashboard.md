# Monitoring Dashboard for Lazy-Loaded Components

This document describes the monitoring dashboard for lazy-loaded components in the TradSys codebase.

## Overview

The monitoring dashboard provides real-time visibility into the state of lazy-loaded components, including memory usage, initialization status, and resource allocation. It helps developers and operators monitor the system's health and identify potential issues.

## Features

1. **Real-Time Memory Monitoring**: Track memory usage and pressure levels across all components.
2. **Component Status Tracking**: Monitor the initialization status and resource usage of individual components.
3. **Automatic Unloading Visibility**: See which components have been automatically unloaded due to memory pressure.
4. **Priority Visualization**: Visualize component priorities and their impact on resource allocation.
5. **Prometheus Integration**: Export metrics to Prometheus for long-term storage and alerting.
6. **Web Interface**: User-friendly web interface for easy monitoring.

## Dashboard Components

### System Overview

The system overview section provides a high-level view of the system's health:

- **Memory Usage**: Total memory usage and percentage of the memory limit.
- **Memory Pressure Level**: Current memory pressure level (Low, Medium, High, Critical).
- **Component Statistics**: Number of total, initialized, and in-use components.
- **Memory Usage Chart**: Historical memory usage over time.

### Component List

The component list section provides detailed information about each component:

- **Name**: Component name.
- **Type**: Component type.
- **Memory Usage**: Current memory usage of the component.
- **Priority**: Component priority (High, Medium, Low).
- **Status**: Initialization status (Initialized, Not Initialized) and usage status (In Use).
- **Idle Time**: Time since the component was last accessed.

## API Endpoints

The dashboard provides several API endpoints for programmatic access:

- **/api/metrics**: Returns all metrics in JSON format.
- **/api/components**: Returns detailed information about all components.
- **/api/system**: Returns system-wide metrics.
- **/api/dashboard**: Returns all dashboard data.
- **/metrics**: Returns metrics in Prometheus format.

## Prometheus Metrics

The dashboard exports the following metrics to Prometheus:

### System Metrics

- **lazy_component_total_memory_usage_bytes**: Total memory usage of lazy-loaded components in bytes.
- **lazy_component_memory_usage_percentage**: Memory usage percentage of lazy-loaded components.
- **lazy_component_count**: Number of lazy-loaded components.
- **lazy_component_initialized_count**: Number of initialized lazy-loaded components.
- **lazy_component_in_use_count**: Number of lazy-loaded components in use.

### Component Metrics

- **lazy_component_memory_usage_bytes**: Memory usage of a lazy-loaded component in bytes.
- **lazy_component_initialized**: Whether a lazy-loaded component is initialized (1) or not (0).
- **lazy_component_in_use**: Whether a lazy-loaded component is in use (1) or not (0).
- **lazy_component_idle_time_seconds**: Idle time of a lazy-loaded component in seconds.

## Usage

### Starting the Dashboard

To start the dashboard, create a `LazyComponentDashboard` instance and start it:

```go
// Create a component coordinator
coordinator := coordination.NewComponentCoordinator(
    coordination.DefaultCoordinatorConfig(),
    logger,
)

// Create the dashboard
dashboard := monitoring.NewLazyComponentDashboard(coordinator, logger)

// Start the dashboard
go func() {
    err := dashboard.Start(":8080")
    if err != nil {
        logger.Error("Failed to start dashboard", zap.Error(err))
    }
}()

// Access the dashboard at http://localhost:8080
```

### Stopping the Dashboard

To stop the dashboard, call the `Stop` method:

```go
// Stop the dashboard
err := dashboard.Stop(ctx)
if err != nil {
    logger.Error("Failed to stop dashboard", zap.Error(err))
}
```

### Accessing the Dashboard

The dashboard is accessible via a web browser at the configured address (e.g., http://localhost:8080).

### Accessing Metrics Programmatically

Metrics can be accessed programmatically via the API endpoints:

```bash
# Get all dashboard data
curl http://localhost:8080/api/dashboard

# Get component information
curl http://localhost:8080/api/components

# Get system metrics
curl http://localhost:8080/api/system

# Get Prometheus metrics
curl http://localhost:8080/metrics
```

## Integration with Monitoring Systems

### Prometheus

The dashboard can be integrated with Prometheus by adding the following to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'lazy_components'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
```

### Grafana

You can create Grafana dashboards using the Prometheus metrics. Here's an example dashboard configuration:

```json
{
  "title": "Lazy Component Dashboard",
  "panels": [
    {
      "title": "Memory Usage",
      "type": "gauge",
      "datasource": "Prometheus",
      "targets": [
        {
          "expr": "lazy_component_memory_usage_percentage",
          "refId": "A"
        }
      ],
      "options": {
        "minValue": 0,
        "maxValue": 100,
        "thresholds": [
          {
            "value": 60,
            "color": "green"
          },
          {
            "value": 75,
            "color": "yellow"
          },
          {
            "value": 85,
            "color": "orange"
          },
          {
            "value": 95,
            "color": "red"
          }
        ]
      }
    },
    {
      "title": "Component Count",
      "type": "stat",
      "datasource": "Prometheus",
      "targets": [
        {
          "expr": "lazy_component_count",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Initialized Components",
      "type": "stat",
      "datasource": "Prometheus",
      "targets": [
        {
          "expr": "lazy_component_initialized_count",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Components In Use",
      "type": "stat",
      "datasource": "Prometheus",
      "targets": [
        {
          "expr": "lazy_component_in_use_count",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Memory Usage Over Time",
      "type": "graph",
      "datasource": "Prometheus",
      "targets": [
        {
          "expr": "lazy_component_memory_usage_percentage",
          "refId": "A"
        }
      ]
    },
    {
      "title": "Component Memory Usage",
      "type": "table",
      "datasource": "Prometheus",
      "targets": [
        {
          "expr": "lazy_component_memory_usage_bytes",
          "refId": "A",
          "instant": true
        }
      ],
      "transformations": [
        {
          "id": "organize",
          "options": {
            "excludeByName": {
              "Time": true
            },
            "renameByName": {
              "Value": "Memory Usage (bytes)",
              "name": "Component",
              "type": "Type"
            }
          }
        }
      ]
    }
  ]
}
```

## Customization

The dashboard can be customized by modifying the HTML, CSS, and JavaScript in the `static` directory. The main files are:

- **index.html**: The main dashboard page.
- **styles.css**: CSS styles for the dashboard.
- **script.js**: JavaScript for the dashboard.

## Best Practices

1. **Regular Monitoring**: Regularly check the dashboard to identify potential issues.
2. **Set Up Alerts**: Configure alerts in Prometheus or Grafana for high memory usage or pressure levels.
3. **Monitor Idle Components**: Keep an eye on components with long idle times that might be candidates for unloading.
4. **Track Memory Pressure**: Monitor memory pressure levels to ensure the automatic unloading system is working effectively.
5. **Check Component Priorities**: Ensure that component priorities are set appropriately based on their importance.

## Conclusion

The monitoring dashboard provides valuable insights into the state of lazy-loaded components, helping developers and operators ensure the system's health and performance. By integrating with Prometheus and Grafana, it can be part of a comprehensive monitoring solution for the TradSys platform.

