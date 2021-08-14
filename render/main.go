package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/unixpickle/essentials"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/model3d/render3d"
)

const (
	ErrorMargin  = 0.01
	FinalVersion = false
)

func main() {
	sun := render3d.NewSphereAreaLight(
		&model3d.Sphere{Center: model3d.XYZ(4, -4, 8), Radius: 2.0},
		render3d.NewColor(60.0),
	)
	groundLight := render3d.NewMeshAreaLight(
		model3d.NewMeshRect(model3d.XYZ(-5, -5, -10), model3d.XYZ(5, 5, -9.9)),
		render3d.NewColor(2.0),
	)
	skyLight := render3d.NewMeshAreaLight(
		model3d.NewMeshRect(model3d.XYZ(-10, -5, 40), model3d.XYZ(10, 50, 40.1)),
		render3d.NewColor(12.0),
	)
	sideLights := render3d.JoinAreaLights(
		render3d.NewMeshAreaLight(
			model3d.NewMeshRect(model3d.XYZ(-5, -5, -5), model3d.XYZ(-4.9, -1, 5)),
			render3d.NewColor(0.6),
		),
		render3d.NewMeshAreaLight(
			model3d.NewMeshRect(model3d.XYZ(4.9, -5, -5), model3d.XYZ(5.0, -1, 5)),
			render3d.NewColor(0.6),
		),
	)
	heartObject := HeartObject()
	scene := render3d.JoinedObject{
		heartObject,
		GroundObject(),
		sun,
		groundLight,
		skyLight,
		sideLights,
	}

	renderer := render3d.BidirPathTracer{
		Camera: render3d.NewCameraAt(model3d.Coord3D{Y: -7, Z: 2},
			model3d.Coord3D{Y: 0, Z: 2}, math.Pi/3.6),
		Light: render3d.JoinAreaLights(sun, groundLight, skyLight, sideLights),

		MaxDepth: 15,
		MinDepth: 3,

		NumSamples: 100,
		MinSamples: 100,

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
	}

	fmt.Println("Ray variance:", renderer.RayVariance(scene, 200, 200, 5))

	res := 100
	stops := 5
	if FinalVersion {
		res = 600
		stops = 40
		renderer.MinSamples = 400
		renderer.NumSamples = 10000
	}

	for i := 0; i < stops; i++ {
		lastLog := time.Now().Unix() - 2
		renderer.LogFunc = func(p, samples float64) {
			curLog := time.Now().Unix()
			if curLog > lastLog+1 {
				lastLog = curLog
				log.Printf("Rendering %.1f%% of stop %d/%d...", p*100, i+1, stops)
			}
		}
		angle := math.Pi * 2 * float64(i) / float64(stops)
		scene[0] = render3d.MatrixMultiply(heartObject, model3d.NewMatrix3Rotation(model3d.Z(1), angle))
		img := render3d.NewImage(res, res)
		renderer.Render(img, scene)
		img.Save(fmt.Sprintf("output%03d.png", i))
	}
}

func HeartObject() render3d.Object {
	f, err := os.Open("../create_mesh/heart.stl")
	essentials.Must(err)
	defer f.Close()
	tris, err := model3d.ReadSTL(f)
	essentials.Must(err)
	mesh := model3d.NewMeshTriangles(tris)
	mesh = mesh.SmoothAreas(0.05, 10)
	mesh = mesh.Rotate(model3d.X(1), -math.Pi/2).Translate(model3d.Z(2))

	collider := model3d.MeshToCollider(mesh)

	obj := &render3d.ColliderObject{
		Collider: collider,
		Material: &render3d.JoinedMaterial{
			Materials: []render3d.Material{
				&render3d.RefractMaterial{
					IndexOfRefraction: 1.3,
					RefractColor:      render3d.NewColor(0.95),
				},
				&render3d.PhongMaterial{
					Alpha:         100.0,
					SpecularColor: render3d.NewColor(0.05),
				},
			},
			Probs: []float64{0.8, 0.2},
		},
	}

	flakes := Flakes(collider)

	return render3d.JoinedObject{obj, flakes}
}

func Flakes(container model3d.Collider) render3d.Object {
	solid := model3d.NewColliderSolid(container)
	mesh := model3d.NewMesh()
	for i := 0; i < 10000; i++ {
		point := model3d.NewCoord3DRandBounds(container.Min(), container.Max())
		if !solid.Contains(point) || container.SphereCollision(point, 0.1) {
			continue
		}
		size := model3d.XYZ(0.03, 0.03, 0.005)
		flake := model3d.NewMeshRect(size.Scale(-1), size)
		flake = flake.Rotate(model3d.NewCoord3DRandUnit(), rand.Float64()*math.Pi/2)
		flake = flake.Translate(point)
		mesh.AddMesh(flake)
	}
	return &render3d.ColliderObject{
		Collider: model3d.MeshToCollider(mesh),
		Material: &render3d.PhongMaterial{
			Alpha:         100.0,
			SpecularColor: render3d.NewColorRGB(1.0, 0.85, 0).Scale(0.5),
			DiffuseColor:  render3d.NewColorRGB(1.0, 0.85, 0).Scale(0.3),
		},
	}
}

func GroundObject() render3d.Object {
	return render3d.JoinedObject{
		// Sky
		&render3d.ColliderObject{
			Collider: model3d.NewRect(model3d.XYZ(-20, 20, -10.0), model3d.XYZ(20, 20.1, 100.0)),
			Material: &render3d.PhongMaterial{
				Alpha:        100.0,
				DiffuseColor: render3d.NewColorRGB(0.5, 0.8, 0.9).Scale(0.7),
			},
		},
		// Ocean
		&render3d.ColliderObject{
			Collider: model3d.NewRect(model3d.XYZ(-20, 4.0, 0.0), model3d.XYZ(20, 20, 0.01)),
			Material: &render3d.PhongMaterial{
				Alpha:        100.0,
				DiffuseColor: render3d.NewColorRGB(0.3, 0.9, 0.9).Scale(0.3),
			},
		},
		// Beach
		&render3d.ColliderObject{
			Collider: model3d.NewRect(model3d.XYZ(-10, -4.0, 0.0), model3d.XYZ(10, 4, 0.01)),
			Material: &render3d.LambertMaterial{
				DiffuseColor: render3d.NewColorRGB(1.0, 0.85, 0.3).Scale(0.4),
			},
		},
	}
}
