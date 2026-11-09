const path = require('path');
const config = require('./webpack.config.js');

module.exports = {
  ...config,
  mode: 'production',
  entry: { goldset: './src/components/goldset/goldset.ts' },
  output: {
    filename: 'goldset.js',
    path: path.resolve(__dirname, '../model_training/goldset/web'),
    publicPath: '/',
  },
};
