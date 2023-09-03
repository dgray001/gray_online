const path = require('path');
const config = require('./webpack.config.js');

module.exports = {
  ... config,
  mode: 'development',
  output: {
    filename: '[name].bundle.js',
    path: path.resolve(__dirname, '/dist'),
    publicPath: '/dist/',
  },
  devServer: {
    compress: true,
    liveReload: true,
    port: 8080,
    proxy: {
      '/api': 'http://127.0.0.1:6807', // gin server
    },
    server: 'http',
  },
}