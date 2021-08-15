# li-heart

This is an attempt to reproduce the 3D heart from Love Island (US).

![Heart spinning animation](heart.gif)

The components of this project are as follows:

 * [fit_outline](fit_outline) - fit a few Bezier curves to the 3D outline of the heart, given a crappy tracing job I did from a screenshot.
 * [create_mesh](create_mesh) - create a 3D mesh from the 2D heart outline. The Bezier curves from `fit_outline` is hard-coded here. The resulting mesh is stored in a file heart.stl.gz, which I have already included in this repo.
 * [render](render) - this creates and renders the actual scene using path tracing. There is a flag at the top of the script called `FinalVersion`. Set this to `true` to export a higher-resolution, much less noisy version with more frames. Rendering takes upwards of 200 CPU hours.
