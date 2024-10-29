SIGN-HMAC
=========

Le traitement ``SIGN-HMAC`` authentifie un fichier à l'aide de HMAC.
Les arguments sont:

* ``signatureFile`` (*string*) - Le chemin du fichier contenant la signature
  du fichier de transfert..
* ``key`` (*string*) - La clé de signature en format base64.
* ``algorithm`` (*string*) - L'algorithme de *hash* utilisé pour créer la
  signature du fichier. Les valeurs acceptées sont ``SHA256``, ``SHA384``,
  ``SHA512`` et ``MD5``.