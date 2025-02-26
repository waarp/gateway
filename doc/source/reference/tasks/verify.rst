VERIFY
======

Le traitement ``VERIFY`` authentifie que le fichier de transfert correspond à la
signature fournie. Les arguments sont :

* ``signatureFile`` (*string*) - Le chemin du fichier contenant la signature
  du fichier de transfert.
* ``method`` (*string*) - La méthode de signature utilisée. Les valeurs
  acceptées sont :

  - ``HMAC-SHA256``
  - ``HMAC-SHA384``
  - ``HMAC-SHA512``
  - ``PGP``
* ``keyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour la validation de signature du fichier. Le type de la clé doit obligatoirement
  être adapté pour la méthode de signature choisie.