##########################
``waarp-gatewayd migrate``
##########################

.. program:: waarp-gatewayd migrate

**Commande**

.. code-block:: shell

   waarp-gatewayd migrate "<VERSION>"


``waarp-gatewayd migrate`` est la commande permettant de migrer une base de
données de sa version actuelle jusqu'à la version donnée, et ce, que cette
migration soit un *upgrade* ou un *downgrade*.

La version doit être un numéro de version de Gateway valide (ex: "0.15.4").
La valeur spéciale *"latest"* est également acceptée, et est la valeur par défaut
utilisé lorsque qu'aucune version n'est spécifiée. Cette valeur spéciale correspond
à la dernière version connue (à savoir la version de l'exécutable Gateway).

**Options**

.. option:: --config <CONFIG_FILE>, -c <CONFIG_FILE>

   Définit le fichier de configuration à utiliser pour se connecter à la base de
   données.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. option:: --dry-run, -d

   Simule la migration sans commit les changements.

   .. warning::

      Cette option **ne fonctionne pas avec MySQL**. MySQL force un commit des
      changements après chaque modification du schéma de données. Par conséquent,
      Gateway ne pourra pas rollback tous les changements une fois la migration
      terminée, et risque de laisser votre base de données dans un état inutilisable.

.. option:: --file <SQL_FILE>, -f <SQL_FILE>

   Si spécifiée, cette option indique à Gateway d'écrire les commandes de la
   migration dans le fichier SQL fourni au lieu de les envoyer directement à la
   base de données.

   Cela est utile dans les cas où Gateway n'aurait pas les droits nécessaires
   pour modifier le schéma de la base de données, ou bien si l'utilisateur
   souhaite inspecter le script de migration avant de l'exécuter. Le fichier
   obtenu peut ensuite être exécuté directement sur la base de données pour
   effectuer la migration.

.. option:: --verbose, -v

   Si présent, augmente la verbosité des logs de la commande. Peut être répété
   jusqu'à trois fois pour augmenter encore plus la verbosité.

**Exemple**

.. code-block:: shell

   waarp-gatewayd migrate -c "gatewayd.ini" "0.15.4"