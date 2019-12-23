package cnfg

import (
	"fmt"
	"os"
	"time"
)

// Complete working example for UnmarshalENV()
func ExampleENV_Unmarshal_simple() {
	// Systems is used to show an example of how to access nested slices.
	type System struct {
		Name   string `env:"name"`
		Signal *[]int `env:"signal"`
	}

	// Config represents your application's environment variable based config inputs.
	// Works with or without pointers.
	type Config struct {
		Debug    bool           `env:"debug"`
		Users    []string       `env:"user"`
		Interval *time.Duration `env:"interval"`
		Systems  []*System      `env:"system"`
	}

	// Make a pointer to your struct with some default data.
	// Maybe this data came from a config file? Using ParseFile()!
	c := &Config{
		Debug: true,
		Users: []string{"me", "you", "them"},
	}

	// Okay set some ENV variables. Pretend you did this in bash.
	os.Setenv("APP_DEBUG", "false")   // turn off debug
	os.Setenv("APP_USER_1", "dad")    // replace "you" with "dad"
	os.Setenv("APP_USER_3", "mom")    // add "mom"
	os.Setenv("APP_INTERVAL", "7m1s") // don't forget the interval!!

	// This adds (creates) systems and signals in sub-slices.
	os.Setenv("APP_SYSTEM_0_NAME", "SysWon")
	os.Setenv("APP_SYSTEM_1_NAME", "SysToo")
	os.Setenv("APP_SYSTEM_1_SIGNAL_0", "12")
	// You can add as many as you like, as long as they are in numerical order.
	os.Setenv("APP_SYSTEM_1_SIGNAL_1", "77")

	fmt.Printf("BEFORE => Debug: %v, Interval: %v, Users: %v, Systems: %v\n",
		c.Debug, c.Interval, c.Users, c.Systems)

	// Make a ENV Decoder with special tag and prefix.
	env := &ENV{Tag: "env", Pfx: "APP"}

	// Run Unmarshal to parse the values into your config pointer:
	ok, err := env.Unmarshal(c)
	if err != nil {
		panic(err)
	}

	// And optionally, do something with the "ok" return value.
	// If you wanted to overwrite ALL configs if ANY env variables are present
	// you could use ok to make and if statement that does that.
	if ok {
		fmt.Println("~ Environment variables were parsed into the config!")
	}

	// If you don't set an env variable for it, it will stay nil.
	// Same for structs and slices.
	if c.Interval == nil {
		fmt.Printf("You forgot to set an interval!")
		return
	}

	fmt.Printf("AFTER => Debug: %v, Interval: %v, Users: %v\n", c.Debug, *c.Interval, c.Users)
	// We added some systems, check them!
	for i, s := range c.Systems {
		fmt.Printf(" %v: System Name: %v, Signals: %v\n", i, s.Name, s.Signal)
	}
	// Output: BEFORE => Debug: true, Interval: <nil>, Users: [me you them], Systems: []
	// ~ Environment variables were parsed into the config!
	// AFTER => Debug: false, Interval: 7m1s, Users: [me dad them mom]
	//  0: System Name: SysWon, Signals: <nil>
	//  1: System Name: SysToo, Signals: &[12 77]
}
