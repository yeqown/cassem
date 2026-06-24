package agent

func DialDelivery(addr string) (DeliveryClient, error) {
	cc, err := dial(addr)
	if err != nil {
		return nil, err
	}

	return NewDeliveryClient(cc), nil
}
