.. _reference-rest-updateconf-export:

Exporter la configuration
===========================

.. http:get:: /api/exportconf

   Exporte tout ou partie de la configuration de la Gateway, au format JSON
   ou YAML, dans le corps de la réponse.

   La structure et le contenu du fichier retourné sont documentés :any:`ici
   <reference-backup-json>`. Il s'agit du même format que celui utilisé par les
   commandes :any:`reference-cmd-waarp-gatewayd-import` et
   :any:`reference-cmd-waarp-gatewayd-export`, ainsi que par le point d'accès
   :any:`reference-rest-updateconf-import`.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqheader Accept: [Optionnel, défaut : ``application/json``] Le format
      souhaité pour le corps de la réponse. Les valeurs acceptées sont
      ``application/json`` (par défaut) et ``application/yaml``.

   :reqheader Targets: [Optionnel, défaut : ``all``] Restreint l'export à une
      ou plusieurs catégories de données. Cet en-tête peut être renseigné
      plusieurs fois afin de spécifier plusieurs catégories.

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

   :resheader Content-Type: Le format du corps de la réponse, correspondant à
      la valeur demandée via l'en-tête ``Accept`` (``application/json`` ou
      ``application/yaml``).

   :statuscode 200: La configuration a été exportée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 403: L'utilisateur n'a pas le droit d'effectuer cette action

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/exportconf HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Accept: application/json
         Targets: rules

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json

         {
           "rules": [
             {
               "name": "my_rule",
               "isSend": false,
               "path": "/rule_path"
             }
           ]
         }
