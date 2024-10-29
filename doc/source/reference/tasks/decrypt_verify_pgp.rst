DECRYPT&VERIFY-PGP
==================

Le traitement ``DECRYPT&VERIFY-PGP`` déchiffre et authentifie un fichier à
l'aide de PGP. Les arguments sont:

* ``outputFile`` (*string*) - Le chemin du nouveau fichier déchiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.plain``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (chiffré) doit
  être conservé ou non après déchiffrage. Par défaut, le fichier chiffré est
  supprimé après déchiffrage.
* ``decryptionPGPKeyName`` (*string*) - Le nom de la clé (privée) PGP de
  déchiffrage. La clé doit exister dans la base de données.
* ``verificationPGPKeyName`` (*string*) - Le nom de la clé (publique) PGP
  d'authentification. La clé doit exister dans la base de données.