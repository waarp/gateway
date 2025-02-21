SIGN-PGP
========

Le traitement ``SIGN-PGP`` signe un fichier à l'aide de PGP.
Les arguments sont:

* ``outputFile`` (*string*) - Le chemin du nouveau fichier contenant la signature
  du fichier de transfert. Doit être différent du chemin du fichier source.
  Par défaut, le chemin sera identique à celui du fichier source avec le suffixe
  ``.sig``.
* ``pgpKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour la signature du fichier. Cette clé doit être de type ``PGP-PRIVATE``.