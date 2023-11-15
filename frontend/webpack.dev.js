const path = require('path');
const fs = require('fs');
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
    allowedHosts: "all",
    compress: true,
    liveReload: true,
    host: '0.0.0.0',
    https: {
        key: fs.readFileSync("cert.key"),
        cert: fs.readFileSync("cert.crt"),
        ca: fs.readFileSync("ca.crt"),
    },
    port: 8080,
    proxy: {
      '/api': 'http://0.0.0.0:6807', // gin server
    },
    server: 'https',
    static: {
      directory: path.resolve(__dirname, '../backend/static/'),
    },
  },
}