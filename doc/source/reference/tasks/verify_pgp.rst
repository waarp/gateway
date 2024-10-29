VERIFY-PGP
==========

Le traitement ``VERIFY-PGP`` authentifie un fichier à l'aide de PGP.
Les arguments sont:

* ``signatureFile`` (*string*) - Le chemin du fichier contenant la signature
  du fichier de transfert.
* ``pgpKeyName`` (*string*) - Le nom de la clé (publique) PGP d'authentification.
  La clé doit exister dans la base de données.