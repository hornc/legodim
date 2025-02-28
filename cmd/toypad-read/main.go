package main

import (
	"context"
	"encoding/binary"
	"log"
	"os"
	"os/signal"

	"github.com/dolmen-go/legodim/tag"
	"github.com/dolmen-go/legodim/toypad"
	"github.com/dolmen-go/legodim/toypadlink"
)

func main() {
	devs, err := toypadlink.List(toypad.VendorID, toypad.ProductID)
	if err != nil {
		log.Fatalln(err)
	}
	dev, err := devs[0]()
	if err != nil {
		log.Fatalln(err)
	}
	defer dev.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// interruption du contexte par SIGTERM (CTRL+C)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		cancel()
	}()

	tp, err := toypad.NewToyPad(ctx, dev, dev)
	if err != nil {
		log.Fatalln(err)
	}

	var count [3]int // count of on each pad
Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case ev := <-tp.Events:
			log.Printf("Event: %v\n", ev)
			if ev.Action == toypad.Add {
				count[int(ev.Pad-1)]++

				//tp.Send(toypad.SetColor(ev.Pad, toypad.RGB{255, 0, 0}))
				tp.Send(toypad.SetColor(ev.Pad, toypad.RGB{77, 197, 172}))

				uid := tag.UID(ev.UID)
				key := uid.Key()

				/*
					tp.Send(toypad.TagRead(ev.Index, 0x23, cb))
					tp.Send(toypad.TagRead(ev.Index, 0x27, cb))
					tp.Send(toypad.TagRead(ev.Index, 0x2B, cb))
				*/
				for p := 7; p < 40; p += 4 {
					p := p
					cb := func(status uint8, b []byte, err error) {
						if len(b) < 4 {
							log.Println(err)
							return
						}
						log.Println("Reading page", p)

						if p == 35 {
							is_vehicle := b[(38-p)*4+1]
							log.Printf("\033[1;33mVehicle?: %d\033[m", is_vehicle)
							if is_vehicle == 1 {
								v := binary.LittleEndian.Uint32(b[(36-p)*4:])
								log.Printf("\033[1;33mVehicle: %d\033[m", v)
							} else {
								c := key.DecryptCharacter(b[(36-p)*4:])
								log.Printf("\033[1;33mCharacter: %d\033[m", c)
							}
						}
					}
					tp.Send(toypad.TagRead(ev.Index, uint8(p), cb))
				}

				// tp.Send(toypad.SetColor(ev.Pad, toypad.RGB{ev.UID[0], ev.UID[1], ev.UID[2]}))
				tp.Send(toypad.Flash(ev.Pad, 5, 10, 50, toypad.RGB{ev.UID[4], ev.UID[5], ev.UID[6]}))
				// tp.Send(toypad.TagModel(ev.Index, nil, nil))
			} else {
				count[int(ev.Pad-1)]--
				if count[int(ev.Pad-1)] == 0 {
					tp.Send(toypad.SetColor(ev.Pad, toypad.RGB{0, 0, 0})) // Switch off
				}
			}
		case err := <-tp.Errors:
			log.Println("Error:", err)
		}
	}

	// Init
	toypad.Wake().Send(dev, 1)
}
