export PATH=$PATH:~/go/bin
sh assets.sh
rm -rf release/kuaichuan.app
rm -rf release/kuaichuan.dmg
fyne package -os darwin -name release/kuaichuan -icon Icon.png -src .
# scp -r static release/kuaichuan.app/Contents/Resources
create-dmg --window-pos 200 120 --window-size 800 400 --app-drop-link 600 185 release/kuaichuan.dmg release/kuaichuan.app
rm -rf release/kuaichuan.app