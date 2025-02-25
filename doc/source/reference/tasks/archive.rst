ARCHIVE
=======

Le traitement ``ARCHIVE`` creé une archive à partir de plusieurs fichier ou
dossiers. Les arguments sont:

* ``files`` (*string*) - Le ou les fichiers/dossiers à ajouter à l'archive sous
  forme d'une liste séparée par des virgules (``,``). Les paternes *shell* sont
  également acceptés pour ajouter plusieurs fichiers d'un coup. Si des dossiers
  sont ajoutés à l'archive, leur arborescence interne est conservée.
* ``compressionLevel`` (*string*) - Le niveau de compression. Doit être compris
  entre 0 (pas de compression) et 9 (compression maximum). Si absent, un niveau
  de compression sera choisi automatiquement.
* ``outputFile`` (*string*) - Le chemin de l'archive. L'extension de cette archive
  déterminera le type d'archive, ainsi que la méthode de compression utilisée.
  Les extensions acceptées sont :

  - ``.zip`` pour créer une archive ZIP avec compression "DEFLATE"
  - ``.tar`` pour créer une archive TAR sans compression
  - ``.tar.gz`` pour créer une archive TAR avec compression "GZIP"
  - ``.tar.bz2`` pour créer une archive TAR avec compression "BZIP2"
  - ``.tar.xz`` pour créer une archive TAR avec compression "XZ"
  - ``.tar.zstd`` pour créer une archive TAR avec compression "ZSTD"
  - ``.tar.zlib`` pour créer une archive TAR avec compression "ZLIB"