package headhunter

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

func (c *Client) GetArea(ctx context.Context, areaID string) (*AreaResponse, error) {
	path := fmt.Sprintf("/areas/%s", areaID)

	data, err := c.get(ctx, path, nil)
	if err != nil {
		c.logger.Error("failed to get area",
			zap.String("area_id", areaID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("get area: %w", err)
	}

	var area AreaResponse
	if err := c.parseResponse(data, &area); err != nil {
		c.logger.Error("failed to parse area response", zap.Error(err))
		return nil, err
	}

	c.logger.Debug("area retrieved",
		zap.String("area_id", areaID),
		zap.String("name", area.Name),
	)

	return &area, nil
}

func (c *Client) GetAllAreas(ctx context.Context) ([]AreaResponse, error) {
	data, err := c.get(ctx, "/areas", nil)
	if err != nil {
		c.logger.Error("failed to get all areas", zap.Error(err))
		return nil, fmt.Errorf("get all areas: %w", err)
	}

	var areas []AreaResponse
	if err := c.parseResponse(data, &areas); err != nil {
		c.logger.Error("failed to parse areas response", zap.Error(err))
		return nil, err
	}

	c.logger.Debug("all areas retrieved", zap.Int("count", len(areas)))

	return areas, nil
}

func (c *Client) GetRussiaCities(ctx context.Context) ([]City, error) {
	// Russia’s id in HH API = 113
	russiaID := "113"

	area, err := c.GetArea(ctx, russiaID)
	if err != nil {
		return nil, fmt.Errorf("get russia area: %w", err)
	}

	cities := extractCities(area)

	c.logger.Info("russia cities retrieved", zap.Int("count", len(cities)))

	return cities, nil
}

// ---- searching  ----

func (c *Client) SearchAreas(ctx context.Context, query string) ([]City, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	// pull full Russia tree once
	root, err := c.GetArea(ctx, "113")
	if err != nil {
		return nil, fmt.Errorf("get russia area: %w", err)
	}

	nodes := flattenCitiesWithPath(root, nil) // []cityNode{id, name, path}

	cityPart, regionPart := splitQuery(q)
	if cityPart == "" {
		return nil, nil
	}

	results := rankMatch(nodes, cityPart, regionPart)

	// keep API surface: return []City
	out := make([]City, 0, len(results))
	for _, n := range results {
		out = append(out, City{ID: n.ID, Name: n.Name})
		if len(out) == 10 {
			break
		}
	}
	return out, nil
}

// ---- helpers ----

type cityNode struct {
	ID   string
	Name string
	Path string
}

func flattenCitiesWithPath(area *AreaResponse, trail []string) []cityNode {
	path := append(trail, area.Name)
	if len(area.Areas) == 0 {
		return []cityNode{{
			ID:   area.ID,
			Name: area.Name,
			Path: strings.Join(path, " / "),
		}}
	}
	var out []cityNode
	for i := range area.Areas {
		out = append(out, flattenCitiesWithPath(&area.Areas[i], path)...)
	}
	return out
}

func rankMatch(nodes []cityNode, cityPart, regionPart string) []cityNode {
	cityNorm := normalizeQuery(cityPart)
	regionNorm := normalizeQuery(regionPart)

	type scored struct {
		cityNode
		score int
	}

	var scoredList []scored
	short := len([]rune(cityNorm)) <= 3

	for _, n := range nodes {
		name := normalizeQuery(n.Name)
		path := normalizeQuery(n.Path)

		// region filter (if region specified but not found in path — skip)
		if regionNorm != "" && !strings.Contains(path, regionNorm) {
			continue
		}

		s := 0

		switch {
		case name == cityNorm:
			s += 100
		case strings.HasPrefix(name, cityNorm):
			// prefix hit
			if short {
				// for very short queries, only exact/prefix count
				s += 70
			} else {
				s += 70
			}
		case !short && strings.Contains(name, cityNorm):
			// substring match allowed only for non-short queries
			s += 40
		default:
			s = 0
		}

		// shorter names win on same score (closer to exact)
		s -= len([]rune(name)) / 50

		if s > 0 {
			scoredList = append(scoredList, scored{n, s})
		}
	}

	sort.SliceStable(scoredList, func(i, j int) bool {
		if scoredList[i].score != scoredList[j].score {
			return scoredList[i].score > scoredList[j].score
		}
		// then by name
		return scoredList[i].Name < scoredList[j].Name
	})

	out := make([]cityNode, len(scoredList))
	for i := range scoredList {
		out[i] = scoredList[i].cityNode
	}
	return out
}

func extractCities(area *AreaResponse) []City {
	var out []City
	var dfs func(a *AreaResponse, trail []string)

	dfs = func(a *AreaResponse, trail []string) {
		if len(a.Areas) == 0 {
			full := strings.Join(append(trail, a.Name), " > ")
			out = append(out, City{ID: a.ID, Name: a.Name, Path: full})
			return
		}
		for i := range a.Areas {
			dfs(&a.Areas[i], append(trail, a.Name))
		}
	}

	dfs(area, nil)
	return out
}

func splitQuery(q string) (cityPart, regionPart string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", ""
	}

	seps := []string{",", "/", "—", "-"}
	for _, sep := range seps {
		if idx := strings.Index(q, sep); idx >= 0 {
			left := strings.TrimSpace(q[:idx])
			right := strings.TrimSpace(q[idx+len(sep):])
			if left != "" {
				return left, right
			}
		}
	}

	parts := strings.Fields(q)
	if len(parts) <= 1 {
		return q, ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func normalizeQuery(s string) string {
	s = strings.ToLower(s)
	repl := []struct{ from, to string }{
		{".", " "}, {"(", " "}, {")", " "},
		{"  ", " "}, {"   ", " "},
	}
	for _, r := range repl {
		s = strings.ReplaceAll(s, r.from, r.to)
	}
	s = strings.Join(strings.Fields(s), " ")
	return s
}
