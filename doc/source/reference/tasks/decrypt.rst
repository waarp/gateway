DECRYPT
=======

Le traitement ``DECRYPT`` déchiffre le fichier de transfert. Les arguments sont :

* ``outputFile`` (*string*) - Le chemin du nouveau fichier déchiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.plain``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (chiffré) doit
  être conservé ou non après déchiffrement. Par défaut, le fichier chiffré est
  supprimé après déchiffrement.
* ``method`` (*string*) - La méthode de chiffrement utilisée. Les valeurs
  acceptées sont :

  - ``AES-CFB``
  - ``AES-CTR``
  - ``AES-OFB``
  - ``PGP``
* ``keyName`` (*string*) - Le nom de la :term:`clé cryptographique` à utiliser
  pour le déchiffrement. Le type de la clé doit obligatoirement être adapté pour
  la méthode de chiffrement choisie.

.. note::
   Il est à noter que le nouveau fichier déchiffré deviendra le nouveau fichier
   cible du transfert une fois le déchiffrement terminé.