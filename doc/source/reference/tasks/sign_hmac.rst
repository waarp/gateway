SIGN-HMAC
=========

Le traitement ``SIGN-HMAC`` signe un fichier à l'aide de HMAC.
Les arguments sont:

* ``outputFile`` (*string*) - Le chemin du nouveau fichier contenant la signature
  du fichier de transfert. Doit être différent du chemin du fichier source.
  Par défaut, le chemin sera identique à celui du fichier source avec le suffixe
  ``.sig``.
* ``hmacKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour la signature du fichier. La clé doit être de type ``HMAC``.
* ``algorithm`` (*string*) - L'algorithme de *hash* utilisé pour créer la
  signature du fichier. Les valeurs acceptées sont ``SHA256``, ``SHA384``,
  ``SHA512`` et ``MD5``.