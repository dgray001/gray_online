const path = require('path');
const config = require('./webpack.config.js');

module.exports = {
  ... config,
  mode: 'production',
  output: {
    filename: '[name].bundle.js',
    path: path.resolve(__dirname, 'public/dist'),
    publicPath: 'dist/',
  },
}