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
* ``decryptionPGPKeyName`` (*string*) - Le nom de la :term:`clé cryptographique`
  à utiliser pour le déchiffrement. Cette clé doit être de type ``PGP-PRIVATE``.
* ``verificationPGPKeyName`` (*string*) - Le nom de la :term:`clé cryptographique`
  à utiliser pour la vérification de signature. Cette clé doit être de type
  ``PGP-PUBLIC`` ou ``PGP-PRIVATE``.