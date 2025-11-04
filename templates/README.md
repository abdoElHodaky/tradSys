# 📋 **TradSys Code Splitting Templates**

This directory contains comprehensive templates and standards for code splitting in TradSys, based on the successful Orders Service pattern.

## 📁 **Template Files**

### **📋 Standards & Guidelines**
- **`CODE_SPLITTING_STANDARDS.md`** - Official standards and patterns
- **`service_split_checklist.md`** - Quality assurance checklist
- **`README.md`** - This file

### **🎯 Component Templates**
- **`service_split_template.go`** - Generic service split template
- **`risk_engine_template.go`** - Risk engine specific template
- **`matching_engine_template.go`** - Matching engine specific template

### **🔧 Tools**
- **`migration_script_template.go`** - Automated migration tool

## 🚀 **Quick Start**

### **1. Choose Your Template**
```bash
# For generic services
cp templates/service_split_template.go your_split_template.go

# For risk engines
cp templates/risk_engine_template.go your_risk_template.go

# For matching engines
cp templates/matching_engine_template.go your_matching_template.go
```

### **2. Replace Placeholders**
```bash
# Replace template placeholders
sed -i 's/{PACKAGE_NAME}/your_package/g' your_template.go
sed -i 's/{COMPONENT}/YourComponent/g' your_template.go
sed -i 's/{component}/your_component/g' your_template.go
```

### **3. Follow the Checklist**
Use `service_split_checklist.md` to ensure quality and compliance.

## 📊 **File Size Limits**

```yaml
Strict Limits:
  - types.go: ≤ 300 lines
  - core.go: ≤ 350 lines  
  - processors.go: ≤ 350 lines
  - validators.go: ≤ 350 lines
  - reference.go: ≤ 20 lines
```

## 🎯 **Success Pattern**

Based on the Orders Service success:
```
Original: internal/orders/service.go (1,084 lines) ❌
Split into: 8 compliant files ✅
  ├── types.go (219 lines)
  ├── core.go (284 lines)
  ├── processors.go (233 lines)
  ├── validators.go (214 lines)
  └── 4 other specialized files
```

## 📋 **Quality Gates**

### **Mandatory**
- [ ] All files under size limits
- [ ] No functionality lost
- [ ] All tests passing
- [ ] Performance maintained

### **Recommended**
- [ ] Code quality improved
- [ ] Documentation comprehensive
- [ ] Error handling enhanced
- [ ] Maintainability improved

---

**Follow these templates to ensure consistency and durability across all TradSys code splits! 🚀**

