SIGN-HMAC
=========

Le traitement ``SIGN-HMAC`` authentifie un fichier à l'aide de HMAC.
Les arguments sont:

* ``signatureFile`` (*string*) - Le chemin du fichier contenant la signature
  du fichier de transfert..
* ``hmacKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour la vérification de signature. La clé doit être de type ``HMAC``.
* ``algorithm`` (*string*) - L'algorithme de *hash* utilisé pour créer la
  signature du fichier. Les valeurs acceptées sont ``SHA256``, ``SHA384``,
  ``SHA512`` et ``MD5``.