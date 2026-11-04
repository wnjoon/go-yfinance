package repair

// Oracle tables for price_shift_test.go. Every expectation was produced by
// running Python yfinance 1.8.0 (e2f6104) on the same inputs:
// PriceHistory._reconcile_price_shifts_with_volume (instrumented to record the
// filtered during/outside samples of each visited range, in visit order),
// _denoise_volume and _estimate_volume_shift_threshold.

var reconcileOracleCases = []reconcileOracleCase{
	{
		name:      "upstream_m1_unitFalse",
		vol:       []float64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019, 1020, 1021, 1022, 1023, 1024, 1025, 1026, 1027, 1028, 1029, 1030, 1031, 1032, 1033, 1034, 1035, 1036, 1037, 1038, 1039},
		ranges:    [][2]int{{20, 40}},
		down:      []int{20},
		threshold: 3.8,
		wantKept:  [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{1020, 1021, 1022, 1023, 1024, 1025, 1026, 1027, 1028, 1029, 1030, 1031, 1032, 1033, 1034, 1035, 1036, 1037, 1038, 1039}, outside: []float64{1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019}},
		},
	},
	{
		name:       "upstream_m1_unitTrue",
		vol:        []float64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019, 1020, 1021, 1022, 1023, 1024, 1025, 1026, 1027, 1028, 1029, 1030, 1031, 1032, 1033, 1034, 1035, 1036, 1037, 1038, 1039},
		ranges:     [][2]int{{20, 40}},
		down:       []int{20},
		threshold:  3.8,
		unitSwitch: true,
		wantKept:   [][2]int{{20, 40}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{1020, 1021, 1022, 1023, 1024, 1025, 1026, 1027, 1028, 1029, 1030, 1031, 1032, 1033, 1034, 1035, 1036, 1037, 1038, 1039}, outside: []float64{1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019}},
		},
	},
	{
		name:      "upstream_m15_unitFalse",
		vol:       []float64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019, 15300, 15315, 15330, 15345, 15360, 15375, 15390, 15405, 15420, 15435, 15450, 15465, 15480, 15495, 15510, 15525, 15540, 15555, 15570, 15585},
		ranges:    [][2]int{{20, 40}},
		down:      []int{20},
		threshold: 3.8,
		wantKept:  [][2]int{{20, 40}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{15300, 15315, 15330, 15345, 15360, 15375, 15390, 15405, 15420, 15435, 15450, 15465, 15480, 15495, 15510, 15525, 15540, 15555, 15570, 15585}, outside: []float64{1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019}},
		},
	},
	{
		name:       "upstream_m15_unitTrue",
		vol:        []float64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019, 15300, 15315, 15330, 15345, 15360, 15375, 15390, 15405, 15420, 15435, 15450, 15465, 15480, 15495, 15510, 15525, 15540, 15555, 15570, 15585},
		ranges:     [][2]int{{20, 40}},
		down:       []int{20},
		threshold:  3.8,
		unitSwitch: true,
		wantKept:   [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{15300, 15315, 15330, 15345, 15360, 15375, 15390, 15405, 15420, 15435, 15450, 15465, 15480, 15495, 15510, 15525, 15540, 15555, 15570, 15585}, outside: []float64{1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019}},
		},
	},
	{
		name:      "upstream_m3_unitFalse",
		vol:       []float64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019, 3060, 3063, 3066, 3069, 3072, 3075, 3078, 3081, 3084, 3087, 3090, 3093, 3096, 3099, 3102, 3105, 3108, 3111, 3114, 3117},
		ranges:    [][2]int{{20, 40}},
		down:      []int{20},
		threshold: 3.8,
		wantKept:  [][2]int{{20, 40}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{3060, 3063, 3066, 3069, 3072, 3075, 3078, 3081, 3084, 3087, 3090, 3093, 3096, 3099, 3102, 3105, 3108, 3111, 3114, 3117}, outside: []float64{1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019}},
		},
	},
	{
		name:       "upstream_m3_unitTrue",
		vol:        []float64{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019, 3060, 3063, 3066, 3069, 3072, 3075, 3078, 3081, 3084, 3087, 3090, 3093, 3096, 3099, 3102, 3105, 3108, 3111, 3114, 3117},
		ranges:     [][2]int{{20, 40}},
		down:       []int{20},
		threshold:  3.8,
		unitSwitch: true,
		wantKept:   [][2]int{{20, 40}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{3060, 3063, 3066, 3069, 3072, 3075, 3078, 3081, 3084, 3087, 3090, 3093, 3096, 3099, 3102, 3105, 3108, 3111, 3114, 3117}, outside: []float64{1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019}},
		},
	},
	{
		name:      "zeros",
		vol:       []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ranges:    [][2]int{{20, 40}},
		down:      []int{20},
		threshold: 3.8,
		wantKept:  [][2]int{{20, 40}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 40}, during: []float64{}, outside: []float64{}},
		},
	},
	{
		name:      "deletion",
		vol:       []float64{1000, 1000, 100, 100, 100, 3000, 3000, 1000, 3000, 1000, 100, 3000, 100, 1000, 100, 1000, 3000, 1000, 1000, 1000, 100, 100, 100, 1000, 100, 3000, 3000, 1000, 3000, 3000},
		ranges:    [][2]int{{4, 13}, {16, 22}, {24, 27}},
		up:        []int{13, 22, 27},
		down:      []int{4, 16, 24},
		threshold: 2,
		wantKept:  [][2]int{{4, 13}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{24, 27}, during: []float64{100, 3000, 3000, 1000, 3000, 1000, 100, 3000, 100, 3000, 1000, 1000, 1000, 100, 100, 100, 3000, 3000}, outside: []float64{1000, 100, 1000, 100, 1000, 1000, 3000, 3000}},
			{rng: [2]int{16, 22}, during: []float64{100, 3000, 3000, 1000, 3000, 1000, 100, 3000, 100, 3000, 1000, 1000, 1000, 100, 100}, outside: []float64{1000, 100, 1000, 100, 1000, 100, 3000, 3000, 1000, 3000, 3000}},
			{rng: [2]int{4, 13}, during: []float64{100, 3000, 3000, 1000, 3000, 1000, 100, 3000, 100}, outside: []float64{1000, 1000, 100, 100}},
		},
	},
	{
		name:      "pooling",
		vol:       []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250, 260, 270, 280, 290, 300, 310, 320, 330, 340, 350, 360, 370, 380, 390, 400},
		ranges:    [][2]int{{2, 5}, {8, 11}, {14, 17}, {20, 23}},
		down:      []int{2, 8, 14, 20},
		threshold: 1e-09,
		wantKept:  [][2]int{{2, 5}, {8, 11}, {14, 17}, {20, 23}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 23}, during: []float64{30, 40, 50, 90, 100, 110, 150, 160, 170, 210, 220, 230}, outside: []float64{240, 250, 260, 270, 280, 290, 300, 310, 320, 330}},
			{rng: [2]int{14, 17}, during: []float64{30, 40, 50, 210, 220, 230, 90, 100, 110, 150, 160, 170}, outside: []float64{60, 70, 80, 180, 190, 200, 120, 130, 140}},
			{rng: [2]int{8, 11}, during: []float64{210, 220, 230, 150, 160, 170, 30, 40, 50, 90, 100, 110}, outside: []float64{180, 190, 200, 120, 130, 140, 60, 70, 80}},
			{rng: [2]int{2, 5}, during: []float64{210, 220, 230, 150, 160, 170, 90, 100, 110, 30, 40, 50}, outside: []float64{180, 190, 200, 120, 130, 140, 60, 70, 80, 10, 20}},
		},
	},
	{
		name:      "pooling_nonpositive",
		vol:       []float64{10, 20, 30, 0, -5, 60, 70, 80, 90, 0, 0, 120, -5, 140, 150, 0, 170, 180, 190, 200, 210, 0, 230, 240, 250, 260, 270, 280, 290, 300, 310, 320, 330, 340, 350, 360, 370, 380, 390, 400},
		ranges:    [][2]int{{2, 5}, {8, 11}, {14, 17}, {20, 23}},
		down:      []int{2, 8, 14, 20},
		threshold: 1e-09,
		wantKept:  [][2]int{{2, 5}, {8, 11}, {14, 17}, {20, 23}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 23}, during: []float64{30, 90, 150, 170, 210, 230}, outside: []float64{240, 250, 260, 270, 280, 290, 300, 310, 320, 330}},
			{rng: [2]int{14, 17}, during: []float64{30, 210, 230, 90, 150, 170}, outside: []float64{60, 70, 80, 180, 190, 200, 120, 140}},
			{rng: [2]int{8, 11}, during: []float64{210, 230, 150, 170, 30, 90}, outside: []float64{180, 190, 200, 120, 140, 60, 70, 80}},
			{rng: [2]int{2, 5}, during: []float64{210, 230, 150, 170, 90, 30}, outside: []float64{180, 190, 200, 120, 140, 60, 70, 80, 10, 20}},
		},
	},
	{
		name:      "exactly10",
		vol:       []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250, 260, 270, 280, 290, 300},
		ranges:    [][2]int{{5, 15}, {20, 25}},
		down:      []int{5, 20},
		threshold: 1e-09,
		wantKept:  [][2]int{{5, 15}, {20, 25}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{20, 25}, during: []float64{60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 210, 220, 230, 240, 250}, outside: []float64{160, 170, 180, 190, 200, 260, 270, 280, 290, 300}},
			{rng: [2]int{5, 15}, during: []float64{60, 70, 80, 90, 100, 110, 120, 130, 140, 150}, outside: []float64{160, 170, 180, 190, 200, 10, 20, 30, 40, 50}},
		},
	},
	{
		name:      "few_during3",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 970, 5000, 5100, 4900},
		ranges:    [][2]int{{17, 20}},
		up:        []int{17},
		threshold: 10,
		wantKept:  [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{17, 20}, during: []float64{5000, 5100, 4900}, outside: []float64{1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 970}},
		},
	},
	{
		name:      "few_during4",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 5000, 5100, 4900, 5050},
		ranges:    [][2]int{{16, 20}},
		up:        []int{16},
		threshold: 10,
		wantKept:  [][2]int{{16, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{16, 20}, during: []float64{5000, 5100, 4900, 5050}, outside: []float64{970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030}},
		},
	},
	{
		name:      "boundary_down_2.0",
		vol:       []float64{1000, 1000, 1000, 2000, 2000, 2000},
		ranges:    [][2]int{{3, 6}},
		down:      []int{3},
		threshold: 2,
		wantKept:  [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{3, 6}, during: []float64{2000, 2000, 2000}, outside: []float64{1000, 1000, 1000}},
		},
	},
	{
		name:      "boundary_up_2.0",
		vol:       []float64{1000, 1000, 1000, 500, 500, 500},
		ranges:    [][2]int{{3, 6}},
		up:        []int{3},
		threshold: 2,
		wantKept:  [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{3, 6}, during: []float64{500, 500, 500}, outside: []float64{1000, 1000, 1000}},
		},
	},
	{
		name:      "boundary_down_1.999",
		vol:       []float64{1000, 1000, 1000, 2000, 2000, 2000},
		ranges:    [][2]int{{3, 6}},
		down:      []int{3},
		threshold: 1.999,
		wantKept:  [][2]int{{3, 6}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{3, 6}, during: []float64{2000, 2000, 2000}, outside: []float64{1000, 1000, 1000}},
		},
	},
	{
		name:      "boundary_up_1.999",
		vol:       []float64{1000, 1000, 1000, 500, 500, 500},
		ranges:    [][2]int{{3, 6}},
		up:        []int{3},
		threshold: 1.999,
		wantKept:  [][2]int{{3, 6}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{3, 6}, during: []float64{500, 500, 500}, outside: []float64{1000, 1000, 1000}},
		},
	},
	{
		name:      "dir_up_0.1_unitFalse",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 100, 104, 98, 101, 99, 103, 97, 102, 100, 101},
		ranges:    [][2]int{{10, 20}},
		up:        []int{10},
		threshold: 3,
		wantKept:  [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{100, 104, 98, 101, 99, 103, 97, 102, 100, 101}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:       "dir_up_0.1_unitTrue",
		vol:        []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 100, 104, 98, 101, 99, 103, 97, 102, 100, 101},
		ranges:     [][2]int{{10, 20}},
		up:         []int{10},
		threshold:  3,
		unitSwitch: true,
		wantKept:   [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{100, 104, 98, 101, 99, 103, 97, 102, 100, 101}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:      "dir_up_10.0_unitFalse",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100},
		ranges:    [][2]int{{10, 20}},
		up:        []int{10},
		threshold: 3,
		wantKept:  [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:       "dir_up_10.0_unitTrue",
		vol:        []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100},
		ranges:     [][2]int{{10, 20}},
		up:         []int{10},
		threshold:  3,
		unitSwitch: true,
		wantKept:   [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:      "dir_up_1.0_unitFalse",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010},
		ranges:    [][2]int{{10, 20}},
		up:        []int{10},
		threshold: 3,
		wantKept:  [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:       "dir_up_1.0_unitTrue",
		vol:        []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010},
		ranges:     [][2]int{{10, 20}},
		up:         []int{10},
		threshold:  3,
		unitSwitch: true,
		wantKept:   [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:      "dir_down_0.1_unitFalse",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 100, 104, 98, 101, 99, 103, 97, 102, 100, 101},
		ranges:    [][2]int{{10, 20}},
		down:      []int{10},
		threshold: 3,
		wantKept:  [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{100, 104, 98, 101, 99, 103, 97, 102, 100, 101}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:       "dir_down_0.1_unitTrue",
		vol:        []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 100, 104, 98, 101, 99, 103, 97, 102, 100, 101},
		ranges:     [][2]int{{10, 20}},
		down:       []int{10},
		threshold:  3,
		unitSwitch: true,
		wantKept:   [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{100, 104, 98, 101, 99, 103, 97, 102, 100, 101}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:      "dir_down_10.0_unitFalse",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100},
		ranges:    [][2]int{{10, 20}},
		down:      []int{10},
		threshold: 3,
		wantKept:  [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:       "dir_down_10.0_unitTrue",
		vol:        []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100},
		ranges:     [][2]int{{10, 20}},
		down:       []int{10},
		threshold:  3,
		unitSwitch: true,
		wantKept:   [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{10000, 10400, 9800, 10100, 9900, 10300, 9700, 10200, 10000, 10100}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:      "dir_down_1.0_unitFalse",
		vol:       []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010},
		ranges:    [][2]int{{10, 20}},
		down:      []int{10},
		threshold: 3,
		wantKept:  [][2]int{},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
	{
		name:       "dir_down_1.0_unitTrue",
		vol:        []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010},
		ranges:     [][2]int{{10, 20}},
		down:       []int{10},
		threshold:  3,
		unitSwitch: true,
		wantKept:   [][2]int{{10, 20}},
		wantSamples: []reconcileOracleSample{
			{rng: [2]int{10, 20}, during: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}, outside: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010}},
		},
	},
}

var denoiseOracleCases = []struct {
	name      string
	vol, want []float64
}{
	{"denoise_n10", []float64{5, 0, 7, 100, 3, 9, 0, 0, 4, 6}, []float64{7, 7, 7, 6, 5, 6, 5, 4, 4, 4}},
	{"denoise_n8", []float64{5, 1, 7, 100, 3, 9, 2, 8}, []float64{6, 5, 6, 5, 7, 7.5, 8, 5.5}},
	{"denoise_n2", []float64{5, 100}, []float64{5, 100}},
	{"denoise_n1", []float64{0}, []float64{0}},
	{"denoise_upstream", []float64{0, 10, 0, 30, 0}, []float64{10, 20, 30, 30, 30}},
}

var thresholdOracleCases = []thresholdOracleCase{
	{name: "thr_upstream", vol: []float64{1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10}, up: []int{20}, down: []int{}, change: 100, interval: "1d", want: 20.8, wantOK: true},
	{name: "thr_trunc", vol: []float64{1, 2, 1, 2, 3, 2, 1, 2, 5, 6, 5, 6, 7, 6}, up: []int{}, down: []int{8}, change: 2, interval: "1d", want: 1.2, wantOK: true},
	{name: "thr_noise_1wk", vol: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 300, 320, 310, 290}, up: []int{}, down: []int{10}, change: 3, interval: "1wk", want: 1.4147233585952874, wantOK: true},
	{name: "thr_noise_1mo", vol: []float64{1000, 1040, 980, 1010, 990, 1030, 970, 1020, 1000, 1010, 300, 320, 310, 290}, up: []int{}, down: []int{10}, change: 3, interval: "1mo", want: 1.4294467171905747, wantOK: true},
	{name: "thr_trunc_edge", vol: []float64{1, 2, 4, 8, 100, 103, 99, 101, 98, 100}, up: []int{}, down: []int{4}, change: 2, interval: "1d", want: 1.5253739132232522, wantOK: true},
	{name: "thr_nosignal", vol: []float64{1000, 1040}, up: []int{}, down: []int{}, change: 3, interval: "1d", want: 0, wantOK: false},
	{name: "thr_zero", vol: []float64{0, 0, 0}, up: []int{}, down: []int{1}, change: 3, interval: "1d", want: 0, wantOK: false},
}
