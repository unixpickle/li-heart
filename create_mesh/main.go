package main

import (
	"compress/gzip"
	"log"
	"math"
	"os"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/model3d/render3d"
	"github.com/unixpickle/model3d/toolbox3d"
)

func main() {
	// Found using the fit_outline program, then refined
	// at the endpoints.
	outline := model2d.JoinedCurve{
		model2d.BezierCurve{
			model2d.XY(0, 293.481332),
			model2d.XY(63.151034, 293.481332),
			model2d.XY(271.922287, 84.131773),
			model2d.XY(301.151311, -1.335062)},
		model2d.BezierCurve{
			model2d.XY(301.417008, -2.069365),
			model2d.XY(321.470461, -60.706464),
			model2d.XY(346.852102, -152.539968),
			model2d.XY(278.451331, -234.937704)},
		model2d.BezierCurve{
			model2d.XY(278.036016, -235.522389),
			model2d.XY(262.631063, -254.079683),
			model2d.XY(179.054775, -332.691285),
			model2d.XY(70.203954, -270.282418)},
		model2d.BezierCurve{
			model2d.XY(69.489261, -269.997112),
			model2d.XY(23.207534, -243.461804),
			model2d.XY(0, -210.997784),
			model2d.XY(0, -210.997784),
		},
	}
	mesh := model2d.NewMesh()
	sideMesh := model2d.CurveMesh(outline, 200)
	mesh.AddMesh(sideMesh)
	mesh.AddMesh(sideMesh.MapCoords(model2d.XY(-1, 1).Mul))
	mesh = mesh.Scale(1.0 / 200.0)

	log.Println("Creating solid...")
	solid := PillowShape(mesh, 0.9, 0.7)

	log.Println("Creating mesh...")
	mesh3d := model3d.MarchingCubesSearch(solid, 0.01, 8)

	log.Println("Saving mesh...")
	f, err := os.Create("heart.stl.gz")
	essentials.Must(err)
	defer f.Close()
	gf := gzip.NewWriter(f)
	model3d.WriteSTL(gf, mesh3d.TriangleSlice())
	defer gf.Close()

	log.Println("Rendering mesh...")
	render3d.SaveRandomGrid("rendering.png", mesh3d, 3, 3, 300, nil)
}

func PillowShape(mesh *model2d.Mesh, maxRadius, heightScale float64) model3d.Solid {
	resolution := 2048
	sdf2d := model2d.MeshToSDF(mesh)
	hm := toolbox3d.NewHeightMap(sdf2d.Min(), sdf2d.Max(), resolution)

	essentials.ReduceConcurrentMap(0, 40000, func() (func(int), func()) {
		localHM := toolbox3d.NewHeightMap(sdf2d.Min(), sdf2d.Max(), resolution)
		sampleCenter := func() (model2d.Coord, float64) {
			for {
				c := model2d.NewCoordRandBounds(sdf2d.Min(), sdf2d.Max())
				c = model2d.ProjectMedialAxis(sdf2d, c, 0, 0)
				dist := sdf2d.SDF(c)
				if dist > 0 {
					return c, dist
				}
			}
		}
		makeSphere := func(_ int) {
			c, dist := sampleCenter()
			localHM.AddSphereFill(c, dist, math.Min(maxRadius, dist))
		}
		aggregate := func() {
			hm.AddHeightMap(localHM)
		}
		return makeSphere, aggregate
	})
	return model3d.TransformSolid(
		&model3d.Matrix3Transform{Matrix: &model3d.Matrix3{1, 0, 0, 0, 1, 0, 0, 0, heightScale}},
		model3d.IntersectedSolid{
			toolbox3d.HeightMapToSolidBidir(hm),
			// Make it smoother around the seam.
			model3d.ProfileSolid(model2d.NewColliderSolid(model2d.MeshToCollider(mesh)), -2.0, 2.0),
		},
	)
}
