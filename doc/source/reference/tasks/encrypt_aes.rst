ENCRYPT-AES
===========

Le traitement ``ENCRYPT-AES`` chiffre un fichier à l'aide de AES.
Les arguments sont:

* ``outputFile`` (*string*) - Le chemin du nouveau fichier chiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.crypt``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (en clair) doit
  être conservé ou non après chiffrage. Par défaut, le fichier clair est
  supprimé après chiffrage.
* ``aesKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour le déchiffrement. La clé doit être de type ``AES``, et doit faire 16, 24 ou
  32 octets de longueur pour utiliser respectivement AES-128, AES-192, ou AES-256.
* ``mode`` (*string*) - Le mode de de fonctionnement de chiffrement par bloc.
  Les valeurs acceptées sont ``CFB``, ``CTR`` et ``OFB``.