# GoFortress Coverage Documentation Images

This directory contains visual assets for the GoFortress Coverage System documentation.

## Required Images

### Badge Examples
- `badge-flat.png` - Example of flat style coverage badge
- `badge-flat-square.png` - Example of flat-square style badge  
- `badge-for-the-badge.png` - Example of for-the-badge style badge

### Dashboard Screenshots
- `dashboard-hero.png` - Main dashboard hero image showing glass-morphism design
- `trend-chart.png` - Interactive trend chart visualization
- `command-palette.png` - Command palette (Cmd+K) interface
- `analytics-dashboard.png` - Advanced analytics dashboard view

### PR Integration
- `pr-comment-comprehensive.png` - Example of comprehensive PR comment template
- `pr-comment-compact.png` - Example of compact PR comment template
- `pr-comment-detailed.png` - Example of detailed PR comment template

### Notification Examples
- `slack-notification.png` - Slack message with rich formatting
- `email-notification.png` - HTML email notification example

### System Architecture
- `architecture.png` - System architecture diagram showing components

## Image Specifications

### Screenshots
- **Format**: PNG with transparency support
- **Resolution**: 1920x1080 or higher for dashboard screenshots
- **DPI**: 144 DPI for crisp display on high-resolution screens
- **Compression**: Optimized for web (under 500KB per image)

### Badges
- **Format**: PNG or SVG (SVG preferred for scalability)
- **Size**: Standard badge dimensions (104x20px for flat style)
- **Colors**: Match GitHub badge color scheme
- **Text**: Use clear, readable fonts

### Diagrams
- **Format**: PNG or SVG
- **Style**: Clean, modern design matching documentation theme
- **Colors**: Use consistent color palette
- **Labels**: Clear, readable text labels

## Generation Guidelines

### Automated Screenshots
Many images can be generated automatically by the coverage system:

```bash
# Generate badge examples
gofortress-coverage badge --coverage 87.2 --style flat --output badge-flat.svg
gofortress-coverage badge --coverage 87.2 --style flat-square --output badge-flat-square.svg
gofortress-coverage badge --coverage 87.2 --style for-the-badge --output badge-for-the-badge.svg

# Generate dashboard (when system is running)
# Navigate to: https://mrz1836.github.io/go-broadcast/
# Take screenshot of main dashboard

# Generate analytics views
# Navigate to: https://mrz1836.github.io/go-broadcast/analytics
# Take screenshots of various analytics views
```

### Manual Screenshots
For PR comments and notification examples:

1. **PR Comments**: Create a test PR and capture screenshots of different comment templates
2. **Slack Notifications**: Set up test webhook and capture notification examples  
3. **Email Notifications**: Configure email notifications and capture HTML examples

### Architecture Diagram
Create using tools like:
- **Draw.io** (diagrams.net) - Free online diagramming tool
- **Lucidchart** - Professional diagramming software
- **Excalidraw** - Hand-drawn style diagrams
- **Mermaid** - Code-based diagram generation

## Image Optimization

Before committing images:

```bash
# Optimize PNG files
pngcrush -reduce -brute *.png

# Optimize JPEG files  
jpegoptim --max=85 *.jpg

# Optimize SVG files
svgo *.svg
```

## Placeholder Images

Until actual screenshots are available, use placeholder images with:
- Correct dimensions and aspect ratios
- Clear labels indicating what the final image should show
- Consistent styling that matches the documentation theme

## Contributing Images

When adding new images:

1. **Name Convention**: Use kebab-case naming (e.g., `pr-comment-example.png`)
2. **Size Optimization**: Optimize file size while maintaining quality
3. **Alt Text**: Ensure corresponding markdown has descriptive alt text
4. **Version Control**: Commit images with descriptive commit messages
5. **Documentation**: Update this README when adding new image requirements

## Future Enhancements

### Planned Image Types
- **Video Demos**: Screen recordings showing system in action
- **Animated GIFs**: Key interactions and workflows
- **Interactive Diagrams**: Clickable architecture diagrams
- **Mobile Screenshots**: Responsive design examples

### Automated Generation
Consider implementing automated screenshot generation as part of the documentation build process:

```yaml
# Example: Automated screenshot generation in CI
- name: Generate Screenshots
  run: |
    # Start local coverage dashboard
    cd .github/coverage
    ./gofortress-coverage dashboard --port 8080 &
    
    # Use puppeteer or similar to capture screenshots
    npx capture-website http://localhost:8080 --output dashboard-hero.png
```

---

## Status

- [ ] Badge examples (flat, flat-square, for-the-badge)
- [ ] Dashboard hero screenshot
- [ ] Trend chart visualization
- [ ] Command palette interface
- [ ] PR comment examples (all templates)
- [ ] Analytics dashboard views
- [ ] Notification examples (Slack, email)
- [ ] Architecture diagram
- [ ] Mobile responsive examples
- [ ] Video demonstrations

## Notes

This directory structure supports the comprehensive documentation created for Phase 8 of the GoFortress Coverage System. Images should be added as the system is deployed and tested, with screenshots captured from actual usage.
