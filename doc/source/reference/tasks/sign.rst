SIGN
=========

Le traitement ``SIGN`` crée une fichier contenant la signature du fichier de
transfert. Les arguments sont :

* ``outputFile`` (*string*) - Le chemin du nouveau fichier contenant la signature
  du fichier de transfert. Doit être différent du chemin du fichier source.
  Par défaut, le chemin sera identique à celui du fichier source avec le suffixe
  ``.sig``.
* ``method`` (*string*) - La méthode de signature à utiliser. Les valeurs
  acceptées sont :

  - ``HMAC-SHA256``
  - ``HMAC-SHA384``
  - ``HMAC-SHA512``
  - ``PGP``
* ``keyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour la signature du fichier. Le type de la clé doit obligatoirement être
  adapté pour la méthode de signature choisie.