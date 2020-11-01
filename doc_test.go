package cnfg

import (
	"fmt"
	"os"
	"time"
)

// Complete working example for ENV.Unmarshal().
func ExampleENV_Unmarshal_simple() { // nolint: funlen
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
		Debug:    true,
		Users:    []string{"me", "you", "them"},
		Interval: nil,
		Systems:  nil,
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

// Complete working example for UnmarshalENV(). Use this method when the "xml"
// struct tag suits your application.
func ExampleUnmarshalENV() { // nolint: funlen
	// Systems is used to show an example of how to access nested slices.
	type System struct {
		Name   string             `xml:"name"`
		Signal []byte             `xml:"signal"`
		Ion    *map[string]string `xml:"ion"`
	}

	// Config represents your application's environment variable based config inputs.
	// Works with or without pointers.
	type Config struct {
		Users []struct {
			Name   string    `xml:"name"`
			Levels []float64 `xml:"level"`
		} `xml:"user"`
		Systems []*System `xml:"system"`
	}

	// Make a pointer to your struct. It may be empty or contain defaults.
	// It may contain nested pointers, structs, maps, slices, etc. It all works.
	c := &Config{}

	// Okay set some ENV variables. Pretend you did this in bash.
	// Setting these will overwrite any existing data. If you set a slice that
	// does not exist, it has to be the _following_ index number. In other words,
	// if your slice is empty, setting APP_USER_1_NAME wont work, you have to start
	// with 0. If your slice len is 2, you can append by setting APP_USER_2_NAME
	os.Setenv("APP_USER_0_NAME", "Tim")
	os.Setenv("APP_USER_0_LEVEL_0", "1")
	os.Setenv("APP_USER_0_LEVEL_1", "13")
	os.Setenv("APP_USER_1_NAME", "Jon")
	os.Setenv("APP_USER_1_LEVEL_0", "1")

	// This adds (creates) systems and signals in sub-slices.
	os.Setenv("APP_SYSTEM_0_NAME", "SysWon")
	os.Setenv("APP_SYSTEM_1_NAME", "SysToo")
	// With []byte you can only pass a string, and it's converted.
	// You cannot access a byte member directly. Do you need to? Let me know!
	os.Setenv("APP_SYSTEM_0_SIGNAL", "123456")
	os.Setenv("APP_SYSTEM_1_SIGNAL", "654321")

	// Maps inside slices! You can nest all you want, but your variable names may get lengthy.
	fmt.Printf("BEFORE => Users: %v, Systems: %v\n", len(c.Users), len(c.Systems))
	os.Setenv("APP_SYSTEM_1_ION_reactor-1", "overload")
	os.Setenv("APP_SYSTEM_1_ION_reactor-2", "underload")

	// Run Unmarshal to parse the values into your config pointer.
	// We ignore "ok" here. You may choose to capture and it do something though.
	_, err := UnmarshalENV(c, "APP")
	if err != nil {
		panic(err)
	}

	fmt.Printf("AFTER  => Users: %v\n", c.Users)

	for i, s := range c.Systems {
		fmt.Printf(" %v: System Name: %v, Signals: %v, Ion: %v\n", i, s.Name, s.Signal, s.Ion)
	}
	// Output:
	// BEFORE => Users: 0, Systems: 0
	// AFTER  => Users: [{Tim [1 13]} {Jon [1]}]
	//  0: System Name: SysWon, Signals: [49 50 51 52 53 54], Ion: <nil>
	//  1: System Name: SysToo, Signals: [54 53 52 51 50 49], Ion: &map[reactor-1:overload reactor-2:underload]
}
