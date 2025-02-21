VERIFY-PGP
==========

Le traitement ``VERIFY-PGP`` authentifie un fichier à l'aide de PGP.
Les arguments sont:

* ``signatureFile`` (*string*) - Le chemin du fichier contenant la signature
  du fichier de transfert.
* ``pgpKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour la vérification de signature. Cette clé doit être de type ``PGP-PUBLIC``
  ou ``PGP-PRIVATE``.