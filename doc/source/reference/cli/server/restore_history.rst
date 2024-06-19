.. _reference-cmd-waarp-gatewayd-restore-history:

##################################
``waarp-gatewayd restore-history``
##################################

.. program:: waarp-gatewayd restore-history

``waarp-gatewayd restore-history`` est une commande permettant de restorer un précédent
*dump* de l'historique de transfert en base de données. Ce dump est généralement
généré par la commande :ref:`reference-cmd-waarp-gatewayd-purge` à l'aide de
l'option ``-e``.

La structure et le contenu du fichier JSON est documenté :any:`ici
<reference-history-dump-json>`.

Cette commande accepte les options suivantes :

.. option:: --config FILE, -c FILE

   Définit le fichier de configuration à utiliser pour accéder à la base de données.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. option:: --source FILE, -s FILE

   :Défaut: entrée standard

   Indique le chemin du fichier source du *dump* à importer. Par défaut, le *dump*
   sera lu depuis l'entrée standard.

.. option:: --dry-run, -d

   Simule l'import sans modifier aucune donnée.

.. option:: --verbose, -v

   Active l'écriture des logs sur la sortie d'erreur.
   Cet argument peut être répété jusqu'à 3 fois pour augmenter la verbosité
   (ex : ``-vvv``).

.. option:: --help, -h

   Affiche l'aide de la commande.
