===========================
Exporter la configuration
===========================

.. program:: waarp-gateway updateconf export

Exporte tout ou partie de la configuration de la Gateway, au format JSON ou
YAML, vers un fichier ou vers la sortie standard.

La structure et le contenu du fichier généré sont documentés :any:`ici
<reference-backup-json>`. Il s'agit du même format que celui utilisé par les
commandes :any:`reference-cmd-waarp-gatewayd-import` et
:any:`reference-cmd-waarp-gatewayd-export`, ainsi que par le point d'accès
:any:`reference-rest-updateconf-import`.

**Commande**

.. code-block:: shell

   waarp-gateway updateconf export

**Options**

.. option:: -f <FILE>, --file=<FILE>

   :Défaut: sortie standard

   Le fichier dans lequel écrire la configuration exportée. Si le nom du
   fichier se termine par *.yml* ou *.yaml*, les données seront exportées au
   format YAML. Sinon, elles seront exportées au format JSON.

.. option:: -t [rules|servers|partners|clients|users|clouds|snmp|authorities|keys|email|filewatchers|all], --target=[rules|servers|partners|clients|users|clouds|snmp|authorities|keys|email|filewatchers|all]

   :Défaut: ``all``

   Restreint l'export à un sous-ensemble de données. Cette option peut être
   renseignée plusieurs fois pour exporter plusieurs sous-ensembles à la fois.

   Les valeurs possibles sont :

   * ``rules``: Règles de transfert.
   * ``servers``: Définitions de serveurs locaux, incluant les comptes locaux
     et informations d'authentification associées.
   * ``partners``: Définitions de partenaires distants, incluant les comptes
     locaux et informations d'authentification associées.
   * ``clients``: Définitions de clients de transfert.
   * ``users``: Identifiants des utilisateurs Waarp Gateway servant à
     l'administration.
   * ``clouds``: Instances de stockage dans le *cloud*.
   * ``snmp``: Configuration du service SNMP (serveur et *monitors*).
   * ``authorities``: Autorités d'authentification.
   * ``keys``: Clés cryptographiques.
   * ``email``: Modèles et informations d'authentification d'envoi d'e-mails.
   * ``filewatchers``: *Filewatchers*.
   * ``all``: Toutes les catégories de données existantes.

|

**Exemple**

.. code-block:: shell

   waarp-gateway updateconf export --file 'config.yaml' --target 'rules'
