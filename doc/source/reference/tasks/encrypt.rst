ENCRYPT
=======

Le traitement ``ENCRYPT`` chiffre le fichier de transfert. Les arguments sont :

* ``outputFile`` (*string*) - Le chemin du nouveau fichier chiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.crypt``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (en clair) doit
  être conservé ou non après chiffrement. Par défaut, le fichier clair est
  supprimé après chiffrement.
* ``method`` (*string*) - La méthode de chiffrement à utiliser. Les valeurs
  acceptées sont :

  - ``AES-CFB``
  - ``AES-CTR``
  - ``AES-OFB``
  - ``PGP``
* ``keyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour le chiffrement. Le type de la clé doit obligatoirement être adapté pour
  la méthode de chiffrement choisie.

.. note::
   Il est à noter que le nouveau fichier chiffré deviendra le nouveau fichier
   cible du transfert une fois le chiffrement terminé.