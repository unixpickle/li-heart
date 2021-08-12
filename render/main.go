package main

import (
	"fmt"
	"math"
	"os"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/model3d/render3d"
)

const ErrorMargin = 0.01

func main() {
	sun := render3d.NewSphereAreaLight(
		&model3d.Sphere{Center: model3d.XYZ(4, -4, 8), Radius: 0.8},
		render3d.NewColor(30.0),
	)
	groundLight := render3d.NewMeshAreaLight(
		model3d.NewMeshRect(model3d.XYZ(-5, -5, -10), model3d.XYZ(5, 5, -9.9)),
		render3d.NewColor(1.0),
	)
	skyLight := render3d.NewMeshAreaLight(
		model3d.NewMeshRect(model3d.XYZ(-5, -5, 10), model3d.XYZ(5, 5, 10.1)),
		render3d.NewColor(1.0),
	)
	sideLights := render3d.JoinAreaLights(
		render3d.NewMeshAreaLight(
			model3d.NewMeshRect(model3d.XYZ(-5, -5, -5), model3d.XYZ(-4.9, -1, 5)),
			render3d.NewColor(0.3),
		),
		render3d.NewMeshAreaLight(
			model3d.NewMeshRect(model3d.XYZ(4.9, -5, -5), model3d.XYZ(5.0, -1, 5)),
			render3d.NewColor(0.3),
		),
	)
	scene := render3d.JoinedObject{
		NewHeart(),
		sun,
		groundLight,
		skyLight,
		sideLights,
	}

	renderer := render3d.BidirPathTracer{
		Camera: render3d.NewCameraAt(model3d.Coord3D{Y: -5, Z: 2},
			model3d.Coord3D{Y: 0, Z: 2}, math.Pi/3.6),
		Light: render3d.JoinAreaLights(sun, groundLight, skyLight, sideLights),

		MaxDepth: 15,
		MinDepth: 3,

		NumSamples: 200,
		MinSamples: 200,

		// Gamma-aware convergence criterion.
		Convergence: func(mean, stddev render3d.Color) bool {
			stddevs := stddev.Array()
			for i, m := range mean.Array() {
				s := stddevs[i]
				if m-3*s > 1 {
					// Oversaturated, so even if the variance
					// is high, this region is stable.
					continue
				}
				if math.Pow(m+s, 1/2.2)-math.Pow(m, 1/2.2) > ErrorMargin {
					return false
				}
			}
			return true
		},

		RouletteDelta: 0.2,

		Antialias: 1.0,
		Cutoff:    1e-4,

		LogFunc: func(p, samples float64) {
			fmt.Printf("\rRendering %.1f%%...", p*100)
		},
	}

	fmt.Println("Ray variance:", renderer.RayVariance(scene, 200, 200, 5))

	img := render3d.NewImage(200, 200)
	renderer.Render(img, scene)
	fmt.Println()
	img.Save("output.png")
}

func NewHeart() render3d.Object {
	f, err := os.Open("../create_mesh/heart.stl")
	essentials.Must(err)
	defer f.Close()
	tris, err := model3d.ReadSTL(f)
	essentials.Must(err)
	mesh := model3d.NewMeshTriangles(tris)
	mesh = mesh.Rotate(model3d.X(1), -math.Pi/2).Translate(model3d.Z(2))

	obj := &render3d.ColliderObject{
		Collider: model3d.MeshToCollider(mesh),
		Material: &render3d.RefractMaterial{
			IndexOfRefraction: 1.4,
			RefractColor:      render3d.NewColor(1.0),
			SpecularColor:     render3d.NewColor(1.0),
		},
		// Material: &render3d.LambertMaterial{
		// 	DiffuseColor: render3d.NewColor(1.0),
		// 	AmbientColor: render3d.NewColor(0.1),
		// },
	}
	return obj
}
