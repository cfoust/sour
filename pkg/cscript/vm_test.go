package cscript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBasicExecution(t *testing.T) {
	vm := NewVM()
	vm.Run(`echo "hello world"`)
}

func TestIntArithmetic(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect int
	}{
		{`(+ 2 3)`, 5},
		{`(- 10 4)`, 6},
		{`(* 3 7)`, 21},
		{`(div 10 3)`, 3},
		{`(mod 10 3)`, 1},
		{`(div 5 0)`, 0},
		{`(+ (+ 1 2) (+ 3 4))`, 10},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestFloatArithmetic(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect float32
	}{
		{`(+f 2.5 3.5)`, 6.0},
		{`(-f 10.0 4.5)`, 5.5},
		{`(*f 2.0 3.0)`, 6.0},
		{`(divf 10.0 4.0)`, 2.5},
	}
	for _, tt := range tests {
		ast := Parse(tt.code)
		result := vm.execBlock(ast)
		if result.GetFloat() != tt.expect {
			t.Errorf("%s = %f, want %f", tt.code, result.GetFloat(), tt.expect)
		}
	}
}

func TestComparisons(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect int
	}{
		{`(= 5 5)`, 1},
		{`(= 5 3)`, 0},
		{`(!= 5 3)`, 1},
		{`(< 3 5)`, 1},
		{`(> 5 3)`, 1},
		{`(<= 5 5)`, 1},
		{`(>= 5 5)`, 1},
		{`(strcmp "hello" "hello")`, 1},
		{`(strcmp "hello" "world")`, 0},
		{`(=s "abc" "abc")`, 1},
		{`(<s "abc" "def")`, 1},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestLogical(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect int
	}{
		{`(! 0)`, 1},
		{`(! 1)`, 0},
		{`(! "")`, 1},
		{`(! "hello")`, 0},
		{`(&& [= 1 1] [= 2 2])`, 1},
		{`(&& [= 1 1] [= 1 2])`, 0},
		{`(|| [= 1 2] [= 2 2])`, 1},
		{`(|| [= 1 2] [= 3 4])`, 0},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestBitwise(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect int
	}{
		{`(& 0xFF 0x0F)`, 0x0F},
		{`(| 0xF0 0x0F)`, 0xFF},
		{`(^ 0xFF 0x0F)`, 0xF0},
		{`(<< 1 8)`, 256},
		{`(>> 256 8)`, 1},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestVariables(t *testing.T) {
	vm := NewVM()

	// Simple assignment
	vm.Run(`x = 42`)
	if vm.GetAlias("x") != "42" {
		t.Errorf("x = %q, want 42", vm.GetAlias("x"))
	}

	// String assignment
	vm.Run(`name = "hello world"`)
	if vm.GetAlias("name") != "hello world" {
		t.Errorf("name = %q, want 'hello world'", vm.GetAlias("name"))
	}

	// Variable lookup
	result := vm.Execute(`$x`)
	if result != 42 {
		t.Errorf("$x = %d, want 42", result)
	}
}

func TestAliasCommand(t *testing.T) {
	vm := NewVM()
	vm.Run(`alias greet [echo "hello"]`)
	// Just verify it doesn't crash
	vm.Run(`greet`)
}

func TestAliasWithArgs(t *testing.T) {
	vm := NewVM()
	vm.Run(`double = [* $arg1 2]`)
	result := vm.Execute(`double 21`)
	if result != 42 {
		t.Errorf("double 21 = %d, want 42", result)
	}
}

func TestIfStatement(t *testing.T) {
	vm := NewVM()

	tests := []struct {
		code   string
		expect int
	}{
		{`if 1 [result 10] [result 20]`, 10},
		{`if 0 [result 10] [result 20]`, 20},
		{`if (= 5 5) [result 1] [result 0]`, 1},
		{`if (strcmp "a" "b") [result 1] [result 0]`, 0},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestLoop(t *testing.T) {
	vm := NewVM()
	vm.Run(`sum = 0; loop i 5 [sum = (+ $sum $i)]`)
	result := vm.Execute(`$sum`)
	// 0+1+2+3+4 = 10
	if result != 10 {
		t.Errorf("loop sum = %d, want 10", result)
	}
}

func TestLoopConcat(t *testing.T) {
	vm := NewVM()
	result := vm.ExecuteStr(`loopconcat i 3 [result $i]`)
	if result != "0 1 2" {
		t.Errorf("loopconcat = %q, want '0 1 2'", result)
	}
}

func TestStringOps(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect string
	}{
		{`concat "hello" "world"`, "hello world"},
		{`concatword "hello" "world"`, "helloworld"},
		{`strlen "hello"`, "5"},
		{`format "%1 is %2" "Go" "great"`, "Go is great"},
	}
	for _, tt := range tests {
		result := vm.ExecuteStr(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %q, want %q", tt.code, result, tt.expect)
		}
	}
}

func TestStrstr(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect int
	}{
		{`strstr "hello world" "world"`, 6},
		{`strstr "hello world" "xyz"`, -1},
		{`strstr "hello" "hello"`, 0},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestListOps(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect string
	}{
		{`at "a b c d" 0`, "a"},
		{`at "a b c d" 2`, "c"},
		{`at "a b c d" 10`, ""},
	}
	for _, tt := range tests {
		result := vm.ExecuteStr(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %q, want %q", tt.code, result, tt.expect)
		}
	}
}

func TestListLen(t *testing.T) {
	vm := NewVM()
	tests := []struct {
		code   string
		expect int
	}{
		{`listlen "a b c"`, 3},
		{`listlen ""`, 0},
		{`listlen "single"`, 1},
	}
	for _, tt := range tests {
		result := vm.Execute(tt.code)
		if result != tt.expect {
			t.Errorf("%s = %d, want %d", tt.code, result, tt.expect)
		}
	}
}

func TestLoopList(t *testing.T) {
	vm := NewVM()
	vm.Run(`count = 0; looplist item "a b c" [count = (+ $count 1)]`)
	result := vm.Execute(`$count`)
	if result != 3 {
		t.Errorf("looplist count = %d, want 3", result)
	}
}

func TestGoCallbacks(t *testing.T) {
	vm := NewVM()

	var captured []string
	vm.AddCommand("testcmd", func(name string, val int) {
		captured = append(captured, name)
	})

	vm.Run(`testcmd "hello" 42`)
	if len(captured) != 1 || captured[0] != "hello" {
		t.Errorf("callback captured = %v, want [hello]", captured)
	}
}

func TestGoCallbackReturn(t *testing.T) {
	vm := NewVM()

	vm.AddCommand("addone", func(n int) int {
		return n + 1
	})

	result := vm.Execute(`addone 5`)
	if result != 6 {
		t.Errorf("addone 5 = %d, want 6", result)
	}
}

func TestGoCallbackStringReturn(t *testing.T) {
	vm := NewVM()

	vm.AddCommand("greet", func(name string) string {
		return "hello " + name
	})

	result := vm.ExecuteStr(`greet "world"`)
	if result != "hello world" {
		t.Errorf("greet world = %q, want 'hello world'", result)
	}
}

func TestInterpolation(t *testing.T) {
	vm := NewVM()
	vm.Run(`x = 5`)
	result := vm.ExecuteStr(`x`)
	// x is an alias, calling it returns its value
	if result != "5" {
		t.Errorf("got %q, want '5'", result)
	}
}

func TestBracketInterpolation(t *testing.T) {
	vm := NewVM()
	vm.Run(`i = 3`)
	// [water@(+ $i 1)] should produce "water4"
	result := vm.ExecuteStr(`result [water@(+ $i 1)]`)
	if result != "water4" {
		t.Errorf("interpolation = %q, want 'water4'", result)
	}
}

func TestCond(t *testing.T) {
	vm := NewVM()
	result := vm.Execute(`cond [= 1 2] [result 10] [= 2 2] [result 20] [result 30]`)
	if result != 20 {
		t.Errorf("cond = %d, want 20", result)
	}
}

func TestCase(t *testing.T) {
	vm := NewVM()
	vm.Run(`x = 2`)
	result := vm.Execute(`case $x 1 [result 10] 2 [result 20] 3 [result 30]`)
	if result != 20 {
		t.Errorf("case = %d, want 20", result)
	}
}

func TestSubstr(t *testing.T) {
	vm := NewVM()
	result := vm.ExecuteStr(`substr "hello world" 6`)
	if result != "world" {
		t.Errorf("substr = %q, want 'world'", result)
	}

	result = vm.ExecuteStr(`substr "hello world" 0 5`)
	if result != "hello" {
		t.Errorf("substr = %q, want 'hello'", result)
	}
}

func TestMathFunctions(t *testing.T) {
	vm := NewVM()

	result := vm.Execute(`abs -5`)
	if result != 5 {
		t.Errorf("abs -5 = %d, want 5", result)
	}

	result = vm.Execute(`min 3 7 1 5`)
	if result != 1 {
		t.Errorf("min = %d, want 1", result)
	}

	result = vm.Execute(`max 3 7 1 5`)
	if result != 7 {
		t.Errorf("max = %d, want 7", result)
	}
}

func TestComments(t *testing.T) {
	vm := NewVM()
	vm.Run(`
		// This is a comment
		x = 42
		// Another comment
		y = 10
	`)
	if vm.Execute(`$x`) != 42 {
		t.Error("x should be 42")
	}
	if vm.Execute(`$y`) != 10 {
		t.Error("y should be 10")
	}
}

func TestMultipleStatements(t *testing.T) {
	vm := NewVM()
	vm.Run(`a = 1; b = 2; c = (+ $a $b)`)
	if vm.Execute(`$c`) != 3 {
		t.Errorf("c = %d, want 3", vm.Execute(`$c`))
	}
}

func TestNestedExpressions(t *testing.T) {
	vm := NewVM()
	result := vm.Execute(`(+ (* 3 4) (- 10 5))`)
	if result != 17 {
		t.Errorf("nested = %d, want 17", result)
	}
}

func TestPush(t *testing.T) {
	vm := NewVM()
	vm.Run(`myvar = "outer"`)

	// Verify myvar is set
	v1 := vm.GetAlias("myvar")
	if v1 != "outer" {
		t.Fatalf("before push, myvar = %q, want 'outer'", v1)
	}

	// In CubeScript, push takes an ident name, pushes value, runs body, pops
	inner := vm.ExecuteStr(`push myvar "inner" [result $myvar]`)
	if inner != "inner" {
		t.Errorf("inside push, myvar = %q, want 'inner'", inner)
	}

	result := vm.GetAlias("myvar")
	// After push, myvar should be restored to "outer"
	if result != "outer" {
		t.Errorf("after push, myvar = %q, want 'outer'", result)
	}
}

func TestWhile(t *testing.T) {
	vm := NewVM()
	vm.Run(`x = 0; while [< $x 5] [x = (+ $x 1)]`)
	if vm.Execute(`$x`) != 5 {
		t.Errorf("while x = %d, want 5", vm.Execute(`$x`))
	}
}

func TestStrReplace(t *testing.T) {
	vm := NewVM()
	result := vm.ExecuteStr(`strreplace "hello world" "world" "go"`)
	if result != "hello go" {
		t.Errorf("strreplace = %q, want 'hello go'", result)
	}
}

func TestAppend(t *testing.T) {
	vm := NewVM()
	vm.Run(`x = "hello"`)
	vm.Run(`append x "world"`)
	if vm.GetAlias("x") != "hello world" {
		t.Errorf("append x = %q, want 'hello world'", vm.GetAlias("x"))
	}
}

func TestIndexOf(t *testing.T) {
	vm := NewVM()
	result := vm.Execute(`indexof "a b c d" "c"`)
	if result != 2 {
		t.Errorf("indexof = %d, want 2", result)
	}
	result = vm.Execute(`indexof "a b c d" "x"`)
	if result != -1 {
		t.Errorf("indexof = %d, want -1", result)
	}
}

// Test against real Sauerbraten cfg files
func TestRealCFGFiles(t *testing.T) {
	// Look for real cfg files in the asset roots
	cfgPaths := []string{
		"../../assets/roots/base/data/default_map_settings.cfg",
		"../../assets/roots/base/data/default_map_models.cfg",
		"../../assets/roots/base/data/game_fps.cfg",
	}

	for _, cfgPath := range cfgPaths {
		absPath, err := filepath.Abs(cfgPath)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			t.Logf("skipping %s (not found)", cfgPath)
			continue
		}

		t.Run(filepath.Base(cfgPath), func(t *testing.T) {
			vm := NewVM()

			// Register the commands that map cfg files use
			var textures []string
			vm.AddCommand("texture", func(type_ string, name string, rot int, xOffset int, yOffset int, scale float32) {
				textures = append(textures, name)
			})
			vm.AddCommand("texturereset", func(limit int) {})
			vm.AddCommand("exec", func(name string) {
				// Don't actually exec sub-files in test
			})
			vm.AddCommand("mmodel", func(name string) {})
			vm.AddCommand("mapmodel", func(rad int, h int, tex int, name string, shadow string) {})
			vm.AddCommand("mapmodelreset", func() {})
			vm.AddCommand("mapsound", func(name string, vol int, maxUses int) {})
			vm.AddCommand("mapsoundreset", func() {})
			vm.AddCommand("registersound", func(name string, vol int) {})
			vm.AddCommand("materialreset", func() {})
			vm.AddCommand("autograss", func(name string) {})
			vm.AddCommand("setshader", func(name string) {})
			vm.AddCommand("setshaderparam", func(args ...string) {})
			vm.AddCommand("skybox", func(name string) {})
			vm.AddCommand("loadsky", func(name string) {})

			// Should not panic
			vm.Run(string(data))

			if filepath.Base(cfgPath) == "default_map_settings.cfg" {
				if len(textures) == 0 {
					t.Error("expected textures to be registered from default_map_settings.cfg")
				} else {
					t.Logf("found %d textures", len(textures))
				}
			}
		})
	}
}

// Test a complex real-world pattern from albatross.cfg
func TestLoopWithInterpolation(t *testing.T) {
	vm := NewVM()

	var textures []string
	vm.AddCommand("texture", func(type_ string, name string, rot int, xOffset int, yOffset int, scale float32) {
		textures = append(textures, type_)
	})

	// This is the actual pattern from albatross.cfg
	vm.Run(`
		loop i 4 [
			texture [water@(+ $i 1)] "golgotha/water2.jpg"
			texture 1 "textures/waterfall.jpg"
		]
	`)

	// Should have 8 textures: 4 water* + 4 "1" entries
	if len(textures) != 8 {
		t.Errorf("expected 8 textures, got %d: %v", len(textures), textures)
	}

	// Check that interpolation produced water1, water2, water3, water4
	expected := []string{"water1", "water2", "water3", "water4"}
	for i, e := range expected {
		if i*2 >= len(textures) {
			break
		}
		if textures[i*2] != e {
			t.Errorf("texture[%d] = %q, want %q", i*2, textures[i*2], e)
		}
	}
}

// Test game_fps.cfg loop pattern
func TestGameFPSPattern(t *testing.T) {
	vm := NewVM()

	// Simplified version of game_fps.cfg
	vm.Run(`
		modenames = "ffa coop teamplay insta"
		loop i (listlen $modenames) [
			mname = (at $modenames $i)
		]
	`)

	// mname should be the last one: "insta"
	result := vm.GetAlias("mname")
	if result != "insta" {
		t.Errorf("mname = %q, want 'insta'", result)
	}
}

func TestThreadSafety(t *testing.T) {
	// Each VM is independent
	vm1 := NewVM()
	vm2 := NewVM()

	vm1.Run(`x = 10`)
	vm2.Run(`x = 20`)

	if vm1.Execute(`$x`) != 10 {
		t.Error("vm1.x should be 10")
	}
	if vm2.Execute(`$x`) != 20 {
		t.Error("vm2.x should be 20")
	}
}
