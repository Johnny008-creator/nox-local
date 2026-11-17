// Line chart with two series that draws in.
function draw(ctx, w, h, t) {
  nox.line(ctx, {
    title: 'Teplota během dne',
    labels: ['6:00', '9:00', '12:00', '15:00', '18:00', '21:00'],
    series: [
      { name: 'Praha', values: [4, 9, 14, 16, 12, 7] },
      { name: 'Brno', values: [5, 11, 16, 18, 13, 8] },
    ],
    t,
  });
}
