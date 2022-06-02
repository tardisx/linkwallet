#!/bin/sh

systemctl daemon-reload
systemctl enable linkwallet
systemctl start linkwallet