package api

func SetDefaults_Devices(devices *Devices) {
	// Add mandatory spice device
	devices.Graphics = []Graphics{
		{
			Port: -1,
			Listen: Listen{
				Type:    "address",
				Address: "0.0.0.0",
			},
		},
	}
	// Use vga as video device, since it is better than cirrus
	// and does not require guest drivers
	var heads uint = 1
	var vram uint = 16384
	devices.Video = []Video{
		{
			Model: VideoModel{
				Type:  "vga",
				Heads: &heads,
				VRam:  &vram,
			},
		},
	}
	// Add mandatory console device
	devices.Consoles = []Console{
		{
			Type: "pty",
		},
	}
}
