# NONAMENO Demo — Go/Ebitengine

Native Go/Ebitengine implementation of the NONAMENO demo. Le projet cible macOS/desktop et Android `arm64-v8a` avec le
même moteur de jeu.

## Fonctionnalités

- champ d’étoiles 3D avec traînées ;
- lettres animées avec interpolations élastiques ;
- scrolltext sinusoïdal ;
- musique YM en PCM stéréo 16 bits à 48 kHz ;
- ressources PNG et YM embarquées dans le binaire.

## Version ordinateur

Prérequis : Go 1.25 ou plus récent.

```sh
go run ./cmd/nonameno
```

Validation :

```sh
go test ./...
go test -race ./...
go vet ./...
```

## Version Android

La configuration fournie utilise Ebitengine/`ebitenmobile` 2.9.11, Android API
36, `minSdk 23`, NDK r28, Gradle 8.11.1, AGP 8.10.1 et Java 17. Elle produit un
APK de débogage pour `arm64-v8a`, adapté aux Google Pixel récents.

Avec un appareil unique branché, déverrouillé et autorisé pour ADB :

```sh
./scripts/run-android.sh
```

Le script génère l’AAR, compile, installe puis lance
`com.olivierh.nonameno/.MainActivity`. Les principaux artefacts sont :

```text
android/app/libs/nonameno.aar
android/app/build/outputs/apk/debug/app-debug.apk
```

Pour compiler sans installer :

```sh
export ANDROID_HOME=/opt/homebrew/share/android-commandlinetools
export JAVA_HOME=/opt/homebrew/opt/openjdk@17

mkdir -p android/app/libs
go run github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.9.11 \
  bind \
  -target android/arm64 \
  -androidapi 23 \
  -javapkg com.olivierh.nonameno \
  -o android/app/libs/nonameno.aar \
  ./mobile
./android/gradlew -p android --console=plain clean assembleDebug
```

## Organisation

```text
game.go                 moteur partagé et ressources embarquées
cmd/nonameno/           lanceur desktop
mobile/                 pont pour ebitenmobile
android/                activité Java et projet Gradle
scripts/run-android.sh  construction, installation et lancement Android
assets/                 polices, logo et musique YM
```

## Performances

Le chemin audio réutilise un tampon mono et écrit directement dans le tampon
PCM fourni par Ebitengine : il n’alloue pas pendant `YMPlayer.Read`. Les glyphes
sont mis en cache, le tri de profondeur réutilise sa mémoire et l’horloge n’est
lue qu’une fois par tick d’animation. Le lanceur desktop évite aussi de
reconstruire une image identique quand la fréquence de l’écran dépasse les 60
mises à jour par seconde.

Mesures indicatives sur Apple M4 Max (`-benchtime=500ms`, moyenne de trois
passes) :

| Chemin mesuré | Avant | Après |
|---|---:|---:|
| lecture YM, 4096 frames | 40,4 µs, 40 960 o, 3 allocs | 11,5 µs, 0 o, 0 alloc |
| mise à jour de 160 tweens | 12,0 µs | 1,36 µs |

Les benchmarks intégrés peuvent être rejoués avec :

```sh
go test -run '^$' -bench . -benchmem
```

## Crédits

- code original : NONAMENO ;
- port Go : ce dépôt.
