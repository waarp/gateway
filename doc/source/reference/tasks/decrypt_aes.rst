DECRYPT-AES
===========

Le traitement ``DECRYPT-AES`` déchiffre un fichier à l'aide de AES.
Les arguments sont:

* ``outputFile`` (*string*) - Le chemin du nouveau fichier déchiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.plain``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (chiffré) doit
  être conservé ou non après déchiffrage. Par défaut, le fichier chiffré est
  supprimé après déchiffrage.
* ``key`` (*string*) - La clé de chiffrement en format base64. La clé doit faire
  16, 24 ou 32 octets de longueur pour sélectionner respectivement AES-128,
  AES-192, ou AES-256.
* ``mode`` (*string*) - Le mode de de fonctionnement de chiffrement par bloc.
  Les valeurs acceptées sont ``CFB``, ``CTR`` et ``OFB``.