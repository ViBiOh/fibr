module.exports = {
  entry: {
    web: __dirname + '/cmd/fibr/static/scripts/thumbnail.js',
  },
  output: {
    path: __dirname + '/cmd/fibr/static/scripts/thumbnail/',
  },
  options: {
    minify: true,
  },
  mode: 'production',
};
