# Micro Services

This repo includes reusable micro services.

## Overview

Services provides a home for real world examples for using Micro v3.

- [blog](blog) - A blog app composed as micro services
- [chat](chat) - An instant messaging or group chat service
- [helloworld](helloworld) - A simple helloworld service
- [messages](messages) - A service for text messages
- [notes](notes) - A note taking service
- [test](test) - A set of sample test services for Micro
- [users](users) - User management and basic auth

## Usage

Install Micro

```
# install micro
go get github.com/micro/micro/v3
```

Run Micro

```
# run the server
micro server
```

Login as an admin

```
# login with user: admin pass: micro
micro login
```

Run a service from source

```
# run the service
micro run github.com/micro/services/helloworld
```

## Contributing

Feel free to contribute by PR and signoff.

## License

[Polyform Strict](https://polyformproject.org/licenses/strict/1.0.0/)

