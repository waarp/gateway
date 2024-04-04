.. _reference-cmd-waarp-gatewayd-change-aes:

########################################
``waarp-gatewayd change-aes-passphrase``
########################################


.. program:: waarp-gatewayd change-aes-passphrase

``waarp-gatewayd change-aes-passphrase`` est la commande permettant de changer
la passphrase AES utilisée par la *gateway* pour chiffrer les mots de passe
distants en base de données.

La commande met à jour tous les mots de passe chiffrés présents dans la base de
données, puis met à jour le fichier de configuration fournit avec la nouvelle
passphrase AES.

Elle accepte les options suivantes :

.. option:: --config FILE, -c FILE

  **REQUIS** Définit le fichier de configuration à utiliser.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. option:: --file, -f

   **REQUIS** Le chemin du fichier contenant la nouvelle passphrase AES.

.. option:: --help, -h

   Affiche l'aide de la commande.
