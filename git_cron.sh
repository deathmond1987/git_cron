#!/usr/bin/env bash
set -xe
systemd_timer=git_cron.timer
systemd_service=git_cron.service
systemd_timer_file=/etc/systemd/system/$systemd_timer
systemd_unit_file=/etc/systemd/system/$systemd_service

get_script_dir () {
   if [ $UID -eq 0 ]; then
       script_dir=/home/$SUDO_USER/.git_cron
   else
       script_dir=$HOME/.git_cron
   fi
)
install_service () {
   get_script_dir
   if -f $systemd_timer_file; then
      echo "systemd time already exits at $systemd_timer_file"
   else echo "[Unit]
Description=timer for git_cron.service

[Timer]
OnCalendar=OnCalendar=*-*-* 2:00:00
Persistent=true
Unit=$systemd_service

[Install]
WantedBy=timers.target" > $systemd_timer_service
   fi

   if -f $systemd_unit_file; then
      echo "systemd unit already exist at $systemd_unit_file"
   else echo "[Unit]
Description=git clone or pull if exist all projects from github
Wants=$systemd_timer
[Service]
Type=oneshot
WorkingDirectory=$script_dir
ExecStart=$script_dir/git_cron.sh

[Install]
WantedBy=multi-user.target" > $systemd_unit_file
   fi

   mkdir -p $script_dir
   wget -O $script_dir/git_cron.sh https://raw.githubusercontent.com/deathmond1987/git_cron/main/git_cron.sh
   chmod 770 $script_dir/git_cron.sh

   systemctl daemon-reload
   systemctl enable --now $systemd_timer
}

remove_service () {
   get_script_dir
   sudo systemctl disable --now $systemd_timer
   sudo systemctl disable --now $systemd_unit
   sudo rm -f $systemd_timer_file
   sudo rm -f $systemd_unit_file
   rm -f $script_dir/git_cron.sh
   if [ -z "$(ls -A $script_dir)" ]; then
       rm -f $script_dir
   else
       echo "$script_dir not empty. refusing to delete folder"
   fi
   sudo systemctl daemon reload
)

while [ "$1" != "" ]; do
   case "$1" in
       -u|--user) shift
                  GH_USER=$1
                  ;;
       -i|--install) install_servive
                  ;;
       -r|--remove) remove_service
                  ;;
       *) echo "unknown arg: $1"
          exit 1 >2
                  ;;
    esac
    shift
done

GH_USER=${GH_USER:=deathmond1987}
PROJECT_LIST=$(curl https://api.github.com/users/$GH_USER/repos\?page\=1\&per_page\=100 | grep -e 'clone_url' | cut -d \" -f 4 | sed '/WSA/d' | xargs -L1)

for project in ${PROJECT_LIST}; do
    project_name=$(echo "${project}" | cut -d'/' -f 5)
    if [ -d ./"${project_name//.git/}" ]; then
        cd ./"${project_name//.git/}"
        git pull
        cd -
    else
        git clone ${project}
    fi
done
