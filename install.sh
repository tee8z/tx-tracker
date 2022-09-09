#!/bin/bash

cwd=$(pwd)

cd /etc/systemd/system

# create service file
echo "Creating service file"
    sudo cat > /etc/systemd/system/tx-tracker.service << EOF
[Unit]
  Description=Slack Bot monitoring base layer bitcoin transactions via mempool.space 
  After=network.target 
  StartLimitIntervalSec=0 

[Service] 
 Type=simple 
 Restart=always 
 RestartSec=1 
 User=root 
 WorkingDirectory=${cwd}/service
 ExecStart=${cwd}/service/tx-tracker

[Install]  
  WantedBy=multi-user.target
EOF

cd $cwd
sudo systemctl enable tx-tracker
sudo systemctl daemon-reload
sudo systemctl start tx-tracker