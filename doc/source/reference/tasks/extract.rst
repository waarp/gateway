EXTRACT
=======

Le traitement ``EXTRACT`` extrait le contenu d'une archive. Les arguments sont:

* ``archive`` (*string*) - Optionnel. L'archive a extraire. Par défaut, le
  fichier de transfert sera utilisé. Les extensions acceptées sont :

  - ``.zip`` pour une archive ZIP avec compression "DEFLATE"
  - ``.tar`` pour une archive TAR sans compression
  - ``.tar.gz`` pour une archive TAR avec compression "GZIP"
  - ``.tar.bz2`` pour une archive TAR avec compression "BZIP2"
  - ``.tar.xz`` pour une archive TAR avec compression "XZ"
  - ``.tar.zstd`` pour une archive TAR avec compression "ZSTD"
  - ``.tar.zlib`` pour une archive TAR avec compression "ZLIB"

* ``outputDir`` (*string*) - Le chemin de destination des fichiers contenus
  dans l'archive. Le chemin doit obligatoirement pointer vers un dossier. Si le
  dossier n'existe pas, il sera créé.

