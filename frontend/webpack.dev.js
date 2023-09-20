const path = require('path');
const config = require('./webpack.config.js');

module.exports = {
  ... config,
  mode: 'development',
  output: {
    filename: '[name].bundle.js',
    path: path.resolve(__dirname, '/dist/'),
    publicPath: '/dist/',
  },
  devServer: {
    compress: true,
    liveReload: true,
    host: '0.0.0.0',
    port: 8080,
    proxy: {
      '/api': 'http://0.0.0.0:6807', // gin server
    },
    server: 'http',
  },
}