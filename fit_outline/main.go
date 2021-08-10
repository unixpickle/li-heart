package main

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/unixpickle/model3d/model2d"
)

func main() {
	mesh := model2d.MustReadBitmap("heart.png", nil).Mesh().SmoothSq(20)
	mesh = mesh.Translate(mesh.Max().Mid(mesh.Min()).Scale(-1))
	mesh.Iterate(func(s *model2d.Segment) {
		if s.Max().X < 0 {
			mesh.Remove(s)
		}
	})
	var cur model2d.Coord
	for _, c := range mesh.VertexSlice() {
		if len(mesh.Find(c)) == 1 && mesh.Find(c)[0][0] == c {
			cur = c
			break
		}
	}
	points := []model2d.Coord{cur}
	for range mesh.SegmentSlice() {
		s := mesh.Find(cur)[0]
		points = append(points, s[1])
		mesh.Remove(s)
		cur = s[1]
	}

	log.Printf("Fitting %d points", len(points))
	fitter := &model2d.BezierFitter{
		NumIters:     200,
		PerimPenalty: 1e-6,
		Momentum:     0.5,
	}

	// These endpoints were found emperically to skip kinks
	// in my flawed sketch and produce a smooth result.
	fracs := []float64{0.0, 0.453, 0.68, 0.90, 1.0}

	// Tangent starts at completely flat for the round tip.
	t1 := &model2d.Coord{X: 1.0}
	var curves []model2d.BezierCurve
	for i, endFrac := range fracs[1:] {
		startFrac := fracs[i]
		start := int(math.Round(startFrac * float64(len(points))))
		end := int(math.Round(endFrac * float64(len(points))))
		log.Printf("Fitting range %d-%d", start, end)
		fit := fitter.FitCubicConstrained(points[start:end], t1, nil, nil)
		curves = append(curves, fit)
		tang := fit[3].Sub(fit[2])
		t1 = &tang
	}

	log.Printf("Fit with %d curves.", len(curves))
	log.Println("SVG:", ToSVGPath(curves))
	log.Println("Code:", ToBezierCode(curves))
	m := model2d.NewMesh()
	for _, c := range curves {
		m.AddMesh(model2d.CurveMesh(c, 100))
	}
	model2d.Rasterize("out.png", m, 1.0)
}

func ToSVGPath(curves []model2d.BezierCurve) string {
	curvesData := fmt.Sprintf("M %f,%f ", curves[0][0].X, curves[0][0].Y)
	for _, curve := range curves {
		curvesData += fmt.Sprintf("C %f,%f %f,%f %f,%f ", curve[1].X, curve[1].Y,
			curve[2].X, curve[2].Y, curve[3].X, curve[3].Y)
	}
	curvesData += "Z"
	return curvesData
}

func ToBezierCode(curves []model2d.BezierCurve) string {
	pieces := make([]string, len(curves))
	for i, curve := range curves {
		points := make([]string, len(curve))
		for i, c := range curve {
			points[i] = fmt.Sprintf("model2d.XY(%f, %f)", c.X, c.Y)
		}
		pieces[i] = "model2d.BezierCurve{" + strings.Join(points, ", ") + "}"
	}
	return "model2d.JoinedCurve{" + strings.Join(pieces, ", ") + "}"
}
