package goth

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

type graphPoint struct {
	x float64
	y float64
}

type graphBar struct {
	x      float64
	y      float64
	width  float64
	height float64
}

func graphComponent(id string, style string, graphType string, values []float64, labels []string, reference string, color string, radius string, height string) templ.Component {
	var builder strings.Builder
	builder.WriteString(`<figure`)
	if id != "" {
		builder.WriteString(` id="`)
		builder.WriteString(html.EscapeString(id))
		builder.WriteString(`"`)
	}
	builder.WriteString(` data-coreui-type="Graph"`)
	if style != "" {
		builder.WriteString(` style="`)
		builder.WriteString(html.EscapeString(style))
		builder.WriteString(`"`)
	}
	builder.WriteString(`>`)
	builder.WriteString(`<svg viewBox="0 0 640 240" width="100%" height="`)
	builder.WriteString(html.EscapeString(defaultGraphHeight(height)))
	builder.WriteString(`" role="img" aria-label="`)
	builder.WriteString(html.EscapeString(defaultGraphType(graphType)))
	builder.WriteString(` graph">`)

	switch {
	case strings.TrimSpace(reference) != "":
		writeGraphPlaceholder(&builder, "Awaiting "+strings.TrimSpace(reference))
	case len(values) == 0:
		writeGraphPlaceholder(&builder, "No graph data")
	default:
		switch defaultGraphType(graphType) {
		case "bar":
			writeBarGraph(&builder, values, color, radius)
		case "area":
			writeAreaGraph(&builder, values, color)
		case "pie":
			writePieGraph(&builder, values, color)
		default:
			writeLineGraph(&builder, values, color)
		}
	}

	builder.WriteString(`</svg>`)
	if len(values) > 0 {
		writeGraphLegend(&builder, withGraphLabels(labels, len(values)), values)
	}
	builder.WriteString(`</figure>`)
	return templ.Raw(builder.String())
}

func defaultGraphType(graphType string) string {
	switch strings.TrimSpace(graphType) {
	case "bar", "area", "pie":
		return strings.TrimSpace(graphType)
	default:
		return "line"
	}
}

func defaultGraphHeight(height string) string {
	height = strings.TrimSpace(height)
	if height == "" {
		return "240px"
	}
	return height
}

func graphFramePath() string {
	return `M 40 20 L 40 200 L 620 200`
}

func writeGraphPlaceholder(builder *strings.Builder, message string) {
	builder.WriteString(`<rect x="16" y="16" width="608" height="208" rx="16" fill="rgba(148, 163, 184, 0.12)"></rect>`)
	builder.WriteString(`<text x="320" y="124" text-anchor="middle" font-size="16" fill="#475569">`)
	builder.WriteString(html.EscapeString(message))
	builder.WriteString(`</text>`)
}

func writeGraphFrame(builder *strings.Builder) {
	builder.WriteString(`<path d="`)
	builder.WriteString(graphFramePath())
	builder.WriteString(`" fill="none" stroke="rgba(148, 163, 184, 0.5)" stroke-width="2"></path>`)
}

func writeLineGraph(builder *strings.Builder, values []float64, color string) {
	writeGraphFrame(builder)
	points := graphPoints(values)
	if len(points) == 0 {
		writeGraphPlaceholder(builder, "No graph data")
		return
	}

	builder.WriteString(`<path d="`)
	builder.WriteString(linePath(points))
	builder.WriteString(`" fill="none" stroke="`)
	builder.WriteString(html.EscapeString(color))
	builder.WriteString(`" stroke-width="4" stroke-linejoin="round" stroke-linecap="round"></path>`)
	writeGraphDots(builder, points, color)
}

func writeAreaGraph(builder *strings.Builder, values []float64, color string) {
	writeGraphFrame(builder)
	points := graphPoints(values)
	if len(points) == 0 {
		writeGraphPlaceholder(builder, "No graph data")
		return
	}

	builder.WriteString(`<path d="`)
	builder.WriteString(areaPath(points))
	builder.WriteString(`" fill="`)
	builder.WriteString(html.EscapeString(color))
	builder.WriteString(`" fill-opacity="0.18"></path>`)
	builder.WriteString(`<path d="`)
	builder.WriteString(linePath(points))
	builder.WriteString(`" fill="none" stroke="`)
	builder.WriteString(html.EscapeString(color))
	builder.WriteString(`" stroke-width="4" stroke-linejoin="round" stroke-linecap="round"></path>`)
	writeGraphDots(builder, points, color)
}

func writeBarGraph(builder *strings.Builder, values []float64, color string, radius string) {
	writeGraphFrame(builder)
	for _, bar := range graphBars(values) {
		builder.WriteString(`<rect x="`)
		builder.WriteString(formatGraphFloat(bar.x))
		builder.WriteString(`" y="`)
		builder.WriteString(formatGraphFloat(bar.y))
		builder.WriteString(`" width="`)
		builder.WriteString(formatGraphFloat(bar.width))
		builder.WriteString(`" height="`)
		builder.WriteString(formatGraphFloat(bar.height))
		builder.WriteString(`" rx="`)
		builder.WriteString(html.EscapeString(radius))
		builder.WriteString(`" ry="`)
		builder.WriteString(html.EscapeString(radius))
		builder.WriteString(`" fill="`)
		builder.WriteString(html.EscapeString(color))
		builder.WriteString(`" fill-opacity="0.9"></rect>`)
	}
}

func writePieGraph(builder *strings.Builder, values []float64, color string) {
	total := 0.0
	filtered := make([]float64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		filtered = append(filtered, value)
		total += value
	}
	if total == 0 {
		writeGraphPlaceholder(builder, "No graph data")
		return
	}

	palette := graphPalette(color)
	startAngle := -90.0
	for i, value := range filtered {
		endAngle := startAngle + (value/total)*360.0
		builder.WriteString(`<path d="`)
		builder.WriteString(describePieSlice(320, 120, 82, startAngle, endAngle))
		builder.WriteString(`" fill="`)
		builder.WriteString(html.EscapeString(palette[i%len(palette)]))
		builder.WriteString(`" stroke="rgba(255, 255, 255, 0.9)" stroke-width="2"></path>`)
		startAngle = endAngle
	}
}

func writeGraphDots(builder *strings.Builder, points []graphPoint, color string) {
	for _, point := range points {
		builder.WriteString(`<circle cx="`)
		builder.WriteString(formatGraphFloat(point.x))
		builder.WriteString(`" cy="`)
		builder.WriteString(formatGraphFloat(point.y))
		builder.WriteString(`" r="4" fill="`)
		builder.WriteString(html.EscapeString(color))
		builder.WriteString(`"></circle>`)
	}
}

func writeGraphLegend(builder *strings.Builder, labels []string, values []float64) {
	builder.WriteString(`<figcaption style="display:flex;flex-wrap:wrap;gap:0.5rem;font-size:0.875rem;">`)
	for i, value := range values {
		builder.WriteString(`<span style="padding:0.25rem 0.5rem;border:1px solid rgba(148, 163, 184, 0.35);border-radius:9999px;">`)
		builder.WriteString(html.EscapeString(labels[i]))
		builder.WriteString(`: `)
		builder.WriteString(html.EscapeString(formatLegendValue(value)))
		builder.WriteString(`</span>`)
	}
	builder.WriteString(`</figcaption>`)
}

func withGraphLabels(labels []string, count int) []string {
	if len(labels) >= count {
		return labels[:count]
	}

	out := make([]string, 0, count)
	out = append(out, labels...)
	for i := len(out); i < count; i++ {
		out = append(out, fmt.Sprintf("Point %d", i+1))
	}
	return out
}

func graphPoints(values []float64) []graphPoint {
	if len(values) == 0 {
		return nil
	}

	min := 0.0
	max := 1.0
	for i, value := range values {
		if i == 0 || value < min {
			min = value
		}
		if i == 0 || value > max {
			max = value
		}
	}
	if min > 0 {
		min = 0
	}
	span := max - min
	if span == 0 {
		span = 1
	}

	width := 620.0 - 40.0
	height := 200.0 - 20.0
	points := make([]graphPoint, 0, len(values))
	for i, value := range values {
		x := 40.0
		if len(values) == 1 {
			x += width / 2
		} else {
			x += width * float64(i) / float64(len(values)-1)
		}
		y := 200.0 - ((value-min)/span)*height
		points = append(points, graphPoint{x: x, y: y})
	}
	return points
}

func graphBars(values []float64) []graphBar {
	if len(values) == 0 {
		return nil
	}

	max := 1.0
	for i, value := range values {
		if i == 0 || value > max {
			max = value
		}
	}
	if max == 0 {
		max = 1
	}

	width := 620.0 - 40.0
	height := 200.0 - 20.0
	gap := 12.0
	barWidth := (width - gap*float64(len(values)-1)) / float64(len(values))
	if barWidth < 12 {
		barWidth = 12
	}

	bars := make([]graphBar, 0, len(values))
	for i, value := range values {
		barHeight := (value / max) * height
		bars = append(bars, graphBar{
			x:      40.0 + float64(i)*(barWidth+gap),
			y:      200.0 - barHeight,
			width:  barWidth,
			height: barHeight,
		})
	}
	return bars
}

func linePath(points []graphPoint) string {
	if len(points) == 0 {
		return ""
	}

	parts := make([]string, 0, len(points))
	for i, point := range points {
		command := "L"
		if i == 0 {
			command = "M"
		}
		parts = append(parts, command+" "+formatGraphFloat(point.x)+" "+formatGraphFloat(point.y))
	}
	return strings.Join(parts, " ")
}

func areaPath(points []graphPoint) string {
	if len(points) == 0 {
		return ""
	}

	return linePath(points) +
		" L " + formatGraphFloat(points[len(points)-1].x) + ` 200` +
		" L " + formatGraphFloat(points[0].x) + ` 200 Z`
}

func describePieSlice(centerX, centerY, radius, startAngle, endAngle float64) string {
	startX, startY := polarToCartesian(centerX, centerY, radius, endAngle)
	endX, endY := polarToCartesian(centerX, centerY, radius, startAngle)
	largeArcFlag := "0"
	if endAngle-startAngle > 180 {
		largeArcFlag = "1"
	}

	return strings.Join([]string{
		"M", formatGraphFloat(centerX), formatGraphFloat(centerY),
		"L", formatGraphFloat(startX), formatGraphFloat(startY),
		"A", formatGraphFloat(radius), formatGraphFloat(radius), "0", largeArcFlag, "0", formatGraphFloat(endX), formatGraphFloat(endY),
		"Z",
	}, " ")
}

func polarToCartesian(centerX, centerY, radius, angleInDegrees float64) (float64, float64) {
	angleInRadians := (angleInDegrees - 90.0) * math.Pi / 180.0
	return centerX + radius*math.Cos(angleInRadians), centerY + radius*math.Sin(angleInRadians)
}

func graphPalette(base string) []string {
	return []string{
		base,
		"rgba(99, 102, 241, 0.82)",
		"rgba(14, 165, 233, 0.78)",
		"rgba(16, 185, 129, 0.78)",
		"rgba(245, 158, 11, 0.8)",
	}
}

func formatLegendValue(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatGraphFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
