ENCRYPT-PGP
===========

Le traitement ``ENCRYPT-PGP`` chiffre un fichier à l'aide de PGP.
Les arguments sont:

* ``outputFile`` (*string*) - Le chemin du nouveau fichier chiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.crypt``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (en clair) doit
  être conservé ou non après chiffrage. Par défaut, le fichier clair est
  supprimé après chiffrage.
* ``pgpKeyName`` (*string*) - Le nom de la clé (publique) PGP de chiffrage.
  La clé doit exister dans la base de données.