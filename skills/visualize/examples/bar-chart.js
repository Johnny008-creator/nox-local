// Bar chart that grows in. Change labels, values and title.
function draw(ctx, w, h, t) {
  nox.bar(ctx, {
    title: 'Návštěvnost webu (tisíce)',
    labels: ['Po', 'Út', 'St', 'Čt', 'Pá', 'So', 'Ne'],
    values: [12, 19, 15, 22, 28, 9, 7],
    t,
  });
}
