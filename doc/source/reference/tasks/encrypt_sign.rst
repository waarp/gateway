ENCRYPT&SIGN
============

Le traitement ``ENCRYPT&SIGN`` combine les traitements ``ENCRYPT`` et ``SIGN``
en une seule tâche. Cependant, contrairement à la tâche ``SIGN``, la signature
du fichier est ici combinée avec les données chiffrées pour le donner que un
seul fichier en sortie. Les arguments sont :

* ``outputFile`` (*string*) - Le chemin du nouveau fichier chiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.crypt``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (en clair) doit
  être conservé ou non après chiffrement. Par défaut, le fichier clair est
  supprimé après chiffrement.
* ``method`` (*string*) - La méthode de chiffrement/signature combinés à utiliser.
  Les valeurs acceptées sont (pour l'heure) :

  - ``PGP``
* ``encryptKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à
  utiliser pour le chiffrement. Le type de la clé doit obligatoirement être
  adapté pour la méthode de chiffrement choisie.
* ``signKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à
  utiliser pour la signature du fichier. Le type de la clé doit obligatoirement
  être adapté pour la méthode de signature choisie.

.. note::
   Il est à noter que le nouveau fichier chiffré et signé deviendra le nouveau
   fichier cible du transfert une fois le chiffrement et la signature terminés.