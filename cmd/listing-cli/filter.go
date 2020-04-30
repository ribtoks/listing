package main

import "log"

func (c *listingClient) filter(data []byte) error {
	ss, err := c.parseSubscribers(data)
	if err != nil {
		return err
	}
	skipped := 0
	for _, s := range ss {
		if c.isSubscriberOK(s) {
			c.printer.Append(s)
		} else {
			skipped += 1
		}
	}
	c.printer.Render()
	log.Printf("Filtered subscribers. count=%v skipped=%v", len(ss), skipped)
	return nil
}
