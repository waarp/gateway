.. _reference-cmd-waarp-gatewayd-export:

#########################
``waarp-gatewayd export``
#########################


.. program:: waarp-gatewayd export

``waarp-gatewayd export`` est une commande qui permet de charger la
configuration de Gateway depuis un fichier JSON.

La structure et le contenu du fichier JSON est documenté :any:`ici
<reference-backup-json>`. Le fichier généré peut être importé avec la commande
:any:`reference-cmd-waarp-gatewayd-import`.

Cette commande accepte les options suivantes :

.. option:: --config FILE, -c FILE

   Définit le fichier de configuration à utiliser.

   Si aucun fichier spécifique n'est fourni avec cet argument, les emplacements
   par défaut suivants sont recherchés (dans cet ordre) :

   * :file:`gatewayd.ini`, relatif au dossier courant (Linux et Windows)
   * :file:`etc/gatewayd.ini`, relatif au dossier courant (Linux)
   * :file:`etc\\gatewayd.ini`, relatif au dossier courant (Windows)
   * :file:`/etc/waarp-gateway/gatewayd.ini` (Linux)
   * :file:`%ProgramData%\\gatewayd.ini` (Windows)

.. option:: --file FILE, -f FILE

   :Défaut: sortie standard

   Indique le chemin vers le fichier dans lequel écrire les données exportées au
   format JSON.

.. option:: -t [rules|servers|partners|users|all], --target [rules|servers|partners|users|all]

   :Défaut: ``all``

   Spécifie un sous-ensemble de données à exporter. Cet argument peut être
   renseigné plusieurs fois pour choisir plusieurs catégories.

   Les valeurs possibles sont :

   * ``rules``: Règles de transfert.
   * ``servers``: Définitions de serveurs locaux, incluant les comptes locaux et
     certificats associés.
   * ``partners``: Définitions de partenaires distants, incluant les comptes
     locaux et certificats associés.
   * ``users``: Identifiants des utilisateurs Waarp Gateway servant à l'administration.
   * ``all``: Toutes les données contenues dans le fichier.

.. option:: --verbose, -v

   Active l'écriture des logs sur la sortie d'erreur.
   Cet argument peut être répété jusqu'à 3 fois pour augmenter la verbosité
   (ex : ``-vvv``).

.. option:: --help, -h

   Affiche l'aide de la commande.
