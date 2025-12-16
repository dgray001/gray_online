const path = require('path');
const fs = require('fs');
const config = require('./webpack.config.js');

module.exports = {
	...config,
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
		port: 8080,
		proxy: [
			{
				context: ['/api'],
				target: 'http://0.0.0.0:6807',
			},
		],
		server: {
			type: 'https',
			options: {
				key: fs.readFileSync("cert.key"),
				cert: fs.readFileSync("cert.crt"),
				ca: fs.readFileSync("ca.csr"),
			},
		},
		static: {
			directory: path.resolve(__dirname, '../backend/static/'),
		},
	},
}