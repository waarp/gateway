DECRYPT&VERIFY
==============

Le traitement ``DECRYPT&VERIFY`` combine les traitements ``DECRYPT`` et ``VERIFY``
en une seule tâche. Le fichier de transfert doit donc contenir une combinaison de
données chiffrées et d'une signature. Les arguments sont :

* ``outputFile`` (*string*) - Le chemin du nouveau fichier déchiffré. Doit être
  différent du chemin du fichier source. Par défaut, le chemin sera identique
  à celui du fichier source avec le suffixe ``.plain``.
* ``keepOriginal`` (*boolean*) - Indique si le fichier source (chiffré) doit
  être conservé ou non après déchiffrement. Par défaut, le fichier chiffré est
  supprimé après déchiffrement.
* ``method`` (*string*) - La méthode de déchiffrement/vérification combinés à
  utiliser. Les valeurs acceptées sont (pour l'heure) :

  - ``PGP``
* ``decryptKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à
  utiliser pour le déchiffrement. Le type de la clé doit obligatoirement être
  adapté pour la méthode de chiffrement choisie.
* ``signKeyName`` (*string*) - Le nom de la :term:`clé cryptographique` à
  utiliser pour la validation de signature du fichier. Le type de la clé doit
  obligatoirement être adapté pour la méthode de signature choisie.

.. note::
   Il est à noter que le nouveau fichier déchiffré et validé deviendra le nouveau
   fichier cible du transfert une fois le déchiffrement et la vérification terminés.