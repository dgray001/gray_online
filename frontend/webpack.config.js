module.exports = {
	entry: {
		index: {
			import: './src/pages/index.ts',
		},
		fiddlesticks: {
			dependOn: 'index',
			import: './src/components/game/games/fiddlesticks/fiddlesticks.ts',
		},
		euchre: {
			dependOn: 'index',
			import: './src/components/game/games/euchre/euchre.ts',
		},
		risq: {
			dependOn: 'index',
			import: './src/components/game/games/risq/risq.ts',
		},
		test_game: {
			dependOn: 'index',
			import: './src/components/game/games/test_game/test_game.ts',
		},
	},
	module: {
		rules: [
			{
				test: /\.scss$/,
				use: [
					'style-loader',
					'css-loader',
					{
						loader: "sass-loader",
						options: {
							api: "modern",
							sassOptions: {
								// Your sass options
							},
						},
					},
				],
				exclude: /node_modules/,
			},
			{
				test: /\.html$/i,
				loader: "html-loader",
				exclude: /node_modules/,
			},
			{
				test: /\.tsx?$/,
				use: 'ts-loader',
				exclude: /node_modules/,
			},
		],
	},
	resolve: {
		extensions: ['.tsx', '.ts', '.js'],
	},
}