package fcm

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

func NewClient(ctx context.Context, credentialFile string) (*messaging.Client, error) {
	opt := option.WithCredentialsFile(credentialFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return client, nil
}

func SendOrderReady(ctx context.Context, client *messaging.Client, deviceToken string, orderID int64) error {
	message := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: "Order Ready",
			Body:  fmt.Sprintf("Your order #%d is ready for pickup!", orderID),
		},
		Data: map[string]string{
			"order_id": fmt.Sprintf("%d", orderID),
		},
	}
	_, err := client.Send(ctx, message)
	return err
}