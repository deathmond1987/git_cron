#!/usr/bin/env bash
set -e
systemd_timer=git_cron.timer
systemd_service=git_cron.service
systemd_timer_file=/etc/systemd/system/$systemd_timer
systemd_unit_file=/etc/systemd/system/$systemd_service
script_dir=/home/$SUDO_USER/.git_cron

help () {
  echo "  downloader for user github projects
  examples:
    $(basename $0) -u GITHUB_USER_NAME
  options :
    -u/--user    -set username
    -i/--install -install script as service
    -r/--remove  -remove installed script
    -t/--time    -set update time in install script. format: 2:00:00
    -h/--help    -help"
}

check_root () {
   if [ $EUID -eq 0 ]; then
       true
   else
       echo "root permissions needed"
       exit 1
   fi
}

install_service () {
   if [ -z $GH_USER ]; then
       echo "You need specify github username"
       echo "script.sh -u YOUR_GITHUB_NAME"
       exit 1
   fi
   check_root
   UPDATE_TIME=${UPDATE_TIME:=2:00:00}
   if [ -f $systemd_timer_file ] ; then
      echo "systemd time already exits at $systemd_timer_file"
   else echo "[Unit]
Description=timer for git_cron.service

[Timer]
OnCalendar=*-*-* $UPDATE_TIME
Persistent=true
Unit=$systemd_service

[Install]
WantedBy=timers.target" > $systemd_timer_file
   fi

   if [ -f $systemd_unit_file ]; then
      echo "systemd unit already exist at $systemd_unit_file"
   else echo "[Unit]
Description=git clone or pull if exist all projects from github
Wants=$systemd_timer
[Service]
Type=oneshot
WorkingDirectory=$script_dir
ExecStart=$script_dir/git_cron.sh -u $GH_USER

[Install]
WantedBy=multi-user.target" > $systemd_unit_file
   fi

   mkdir -p $script_dir
   wget -O $script_dir/git_cron.sh https://raw.githubusercontent.com/deathmond1987/git_cron/main/git_cron.sh
   sudo chmod 770 $script_dir/git_cron.sh
   systemctl daemon-reload
   systemctl enable --now $systemd_timer
   systemctl enable --now $systemd_service
}

remove_service () {
    check_root
    systemctl disable --now $systemd_timer || true
    systemctl disable --now $systemd_service || true
    rm -f $systemd_timer_file || true
    rm -f $systemd_unit_file || true
    rm -f $script_dir/git_cron.sh || true
    if [ -z "$(ls -A $script_dir)" ]; then
        rm -rf $script_dir || true
    else
        echo "$script_dir not empty. refusing to delete folder"
    fi
    systemctl daemon-reload
}

get_github () {
    if [ -z $GH_USER ]; then
        echo "You need specify github username"
        echo "script.sh -u YOUR_GITHUB_NAME"
        exit 1
    fi
    GH_USER=${GH_USER:=deathmond1987}
    PROJECT_LIST=$(curl https://api.github.com/users/$GH_USER/repos\?page\=1\&per_page\=100 | grep -e 'clone_url' | cut -d \" -f 4 | sed '/WSA/d' | xargs -L1)
    for project in ${PROJECT_LIST}; do
        project_name=$(echo "${project}" | cut -d'/' -f 5)
        echo "[$project_name] start:"
        if [ -d ./"${project_name//.git/}" ]; then
            cd ./"${project_name//.git/}"
            git pull
            cd -
        else
            git clone ${project}
        fi
        echo "[$project_name] done."
    done
}

main () {
    while [ "$1" != "" ]; do
        case "$1" in
           -u|--user)    shift
                         GH_USER=$1
                         ;;
           -i|--install) install_service
                         exit 0
                         ;;
           -r|--remove)  remove_service
                         exit 0
                         ;;
           -h|--help)    help
                         exit 0
                         ;;
           -t|--time)    shift
                         UPDATE_TIME=$1
                         ;;
           *)            echo "unknown arg: $1"
                         exit 1
                         ;;
        esac
        shift
    done
    get_github
}

main "$@"
