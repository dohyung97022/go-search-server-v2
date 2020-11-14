package main

import (
	"context"
	firebasePackage "firebase.google.com/go"
	authPackage "firebase.google.com/go/auth"
	"google.golang.org/api/option"
)

//Firebase ----------------------------------------------------------
type Firebase struct {
	getToken firebaseGetToken
	getStr firebaseGetStr
	app *firebasePackage.App
	auth *authPackage.Client
	context *context.Context
}
//firebase constructor
func newFirebase(serverReaderContext *context.Context)(firebase Firebase, err error){
	credentialClientOption := option.WithCredentialsFile("../firebaseCredential.json")
	app, err := firebasePackage.NewApp(*serverReaderContext, nil, credentialClientOption)
	if err != nil {
		logger.Printf(err.Error())
		return firebase, err
	}
	auth, err := app.Auth(*serverReaderContext)
	if err != nil {
		logger.Printf(err.Error())
		return firebase, err
	}
	firebase.app = app
	firebase.auth = auth
	firebase.context = serverReaderContext

	firebase.getToken.app = app
	firebase.getToken.auth = auth
	firebase.getToken.context = serverReaderContext
	return firebase, nil
}

//firebase.getToken
type firebaseGetToken struct {
	app *firebasePackage.App
	auth *authPackage.Client
	context *context.Context
}
//firebase.getToken.fromTokenStr
func (firebase *firebaseGetToken) fromTokenStr (tokenStr string) (resToken *authPackage.Token, err error) {
	resToken, err = firebase.auth.VerifyIDToken(*firebase.context, tokenStr)
	if err != nil {
		logger.Printf(err.Error())
		return nil, err
	}
	logger.Printf("Verified ID token: %v\n", resToken)
	return resToken, nil
}

//firebase.getStr
type firebaseGetStr struct {
}
//Mysql.getStr.UIDFromToken
func (*firebaseGetStr) UIDFromToken (token *authPackage.Token)(UID string){
	return token.UID
}