#!/bin/bash

# Import vars from .env file
set -a; source .env; set +a

workingDirectory="${WORKING_DIRECTORY:-$HOME}"
hostname="${HOSTNAME:-$(hostname -s)}"
echo "Running setup with working directory set to $workingDirectory and a hostname of $hostname"

# Install updates and needed packages
sudo apt-get update
sudo apt-get -qy upgrade
sudo apt-get -qy install mpv unattended-upgrades gpg prometheus-node-exporter wireguard
sudo apt-get -qy autoclean
if [ ! -f "$HOME/.ssh/id_ed25519" ]; then
    echo "Generating SSH key"
    ssh-keygen -t ed25519 -C "$hostname" -q -f "$HOME/.ssh/id_ed25519" -N ""
fi

# Enable the ability to turns the screen off
if ! grep -q " vc4.force_hotplug=1" /boot/firmware/cmdline.txt ; then
  echo "Modifying /boot/firmware/cmdline.txt"
  sudo bash -c "echo -n ' vc4.force_hotplug=1' >> /boot/firmware/cmdline.txt"
fi

if [ ! -f /usr/local/bin/screenoff.sh ]; then
  echo "Installing screen off helper script"
  cat << EOF >> screenoff.sh
#!/bin/bash
export WAYLAND_DISPLAY=wayland-1
export XDG_RUNTIME_DIR=/run/user/1000
/usr/bin/wlr-randr --output HDMI-A-1 --off
EOF
  chmod +x screenoff.sh
  sudo mv screenoff.sh /usr/local/bin/screenoff.sh
fi

if [ ! -f /usr/local/bin/media-sync.sh ]; then
  echo "Installing media sync helper script"
  cat << EOF >> media-sync.sh
#!/bin/bash
rclone sync mediasync:album/HomePictureFrame $workingDirectory/Pictures
# rclone sync mediasync:media/by-year $workingDirectory/Pictures
EOF
  chmod +x media-sync.sh
  sudo mv media-sync.sh /usr/local/bin/media-sync.sh
fi

# Setup crons for turning the screen off at night, rebooting in the morning, and file sync
cat << EOF >> crons
# m h  dom mon dow   command
00 23 * * * /usr/local/bin/screenoff.sh
00 06 * * * sudo shutdown -r now
5 * * * * /usr/local/bin/media-sync.sh
EOF
crontab crons
rm crons

# Configure raspi
sudo raspi-config nonint do_change_locale en_US.UTF-8 UTF-8
sudo raspi-config nonint do_expand_rootfs
sudo raspi-config nonint do_wifi_country US
# sudo raspi-config nonint do_hostname $hostname
sudo raspi-config nonint do_boot_behaviour B4
sudo raspi-config nonint do_boot_splash 0
sudo raspi-config nonint do_blanking 0

# Setup WIFI if provided
if [ -z "$WIRELESS_SSID" ]; then
  sudo raspi-config nonint do_wifi_ssid_passphrase "$WIRELESS_SSID" "$WIRELESS_PASSPHRASE"
fi

# wlr-randr --output HDMI-A-1 --transform 180
# -> add 'display_rotate=2'

chmod +x memoryShare
sudo mv memoryShare /usr/local/bin

# Install and enable services
sed -i "s|@@MEDIA_FOLDERS@@|${MEDIA_DIRECTORY}|" media-controller.service
sudo mv media-controller.service /etc/systemd/system/
sudo mv media-player.service /etc/systemd/system/
# sudo mv media-sync.service /etc/systemd/system/

sudo systemctl daemon-reload
sudo systemctl enable media-player.service
sudo systemctl enable media-controller.service
# sudo systemctl enable media-sync.service

sudo systemctl start media-player.service
sudo systemctl start media-controller.service
echo Install and setup rclone with 'sudo -v ; curl https://rclone.org/install.sh | sudo bash'
# echo then run 'sudo systemctl start media-sync.service'

# Disable image wallpaper in favor of a soild color
DISPLAY=:0 pcmanfm --wallpaper-mode=color

# Set desktop color to black and hide default icons
sed -e "/^desktop_bg=/c\desktop_bg=#000000" \
    -e "/^show_trash=/c\show_trash=0" \
    -e "/^show_mounts=/c\show_mounts=0" \
    -i  ~/.config/pcmanfm/LXDE-pi/desktop-items-0.conf


# Simplify and auto hide the menu bar, disable notifications
if ! grep -q " autohide=true" ~/.config/wf-panel-pi.ini ; then
  echo "Modifying ~/.config/wf-panel-pi.ini"
  cat << EOF >> ~/.config/wf-panel-pi.ini
autohide=true
minimal_height=0
notify_enable=false
autohide_duration=5
widgets_left=smenu
widgets_right=clock
EOF
fi

# Sets UI to dark theme
if [ ! -f /etc/environment ]; then
  echo "Settting dark theme"
  cat << EOF >> /etc/environment
GTK_THEME=Adwaita-dark
EOF
fi
