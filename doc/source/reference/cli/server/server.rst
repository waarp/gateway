.. _ref-gatewayd-server:

#########################
``waarp-gatewayd server``
#########################


.. program:: waarp-gatewayd server

``waarp-gatewayd server`` est la commande de lancement de la passerelle.

Elle accepte les options suivantes :

.. option:: --config FILE, -c FILE

   Définit le fichier de configuration à utiliser.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. option:: --update, -u

   Met à jour le fichier de configuration renseigné par le paramètre
   :option:`--config`.

   Le paramètre :option:`--config` doit être renseigné.

.. option:: --create, -n

   Créé un nouveau fichier de configuration à l'emplacement indiqué par le
   paramètre :option:`--config`.

   Le paramètre :option:`--config` doit être renseigné.

.. option:: --instance NAME, -i NAME

   Le nom unique de l'instance. Lorsque Waarp Gateway fonctionne en grappe, ce nom
   sert à différencier les différentes instances entre elles. Ce paramètre n'est
   pas nécessaire si Waarp Gateway ne fonctionne pas en grappe. Il est en revanche
   **obligatoire** si la Gateway fait partie d'une grappe, et il doit également
   être unique au sein de la grappe.

.. option:: --help, -h

   Affiche l'aide de la commande.
