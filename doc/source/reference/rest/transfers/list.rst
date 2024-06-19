Lister les transferts
=====================


.. http:get:: /api/transfers

   Renvoie une liste des transferts remplissant les critères donnés en
   paramètre de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les transferts seront triés.
      Valeurs possibles : ``start+``, ``start-``, ``id+``, ``id-``, ``status+``,
      ``status-``.
      *(défaut: start+)*
   :type sort: string
   :param remote: Filtre uniquement les transferts avec le partenaire renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs partenaires.
   :type remote: int
   :param account: Filtre uniquement les transferts avec le compte renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs comptes.
   :type account: int
   :param rule: Filtre uniquement les transferts avec la règle renseignée.
      Peut être renseigné plusieurs fois pour filtrer plusieurs règles.
   :type rule: int
   :param status: Filtre uniquement les transferts ayant le statut renseigné.
      Valeurs possibles : ``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED`` ou ``ERROR``.
      Peut être renseigné plusieurs fois pour filtrer plusieurs status.
   :type status: string
   :param start: Filtre uniquement les transferts dont la date est ultérieure à
      celle renseignée.
   :type start: date

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array transfers: La liste des transferts demandés
   :resjsonarr number id: L'identifiant local du transfert
   :resjsonarr string remoteID: L'identifiant global du transfert
   :resjsonarr bool isServer: Indique si Gateway est agit en tant que serveur
     (``true``) ou en tant que client (``false``)
   :resjsonarr bool isSend: Indique si le transfert est un envoi (``true``) ou une
     réception (``false``)
   :resjsonarr string rule: Le nom de la règle de transfert
   :resjsonarr string requester: Le nom du compte ayant demandé le transfert
   :resjsonarr string requested: Le nom du serveur/partenaire auquel le transfert a été demandé
   :resjsonarr string protocol: Le protocole utilisé pour effectuer le transfert
   :resjsonarr string srcFilename: Le nom du fichier source.
   :resjsonarr string destFilename: Le nom du fichier destination.
   :resjsonarr string localFilepath: Le chemin du fichier sur le disque local
   :resjsonarr string remoteFilepath: Le chemin du fichier sur le partenaire distant
   :resjsonarr number filesize: La taille du fichier (-1 si inconnue)
   :resjsonarr date start: La date de début du transfert
   :resjsonarr date stop: La date de fin du transfert (si le transfert est terminé)
   :resjsonarr string status: Le statut actuel du transfert (valeurs possibles:
     ``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED``, ``ERROR``, ``DONE``
     ou ``CANCELLED``)
   :resjsonarr string step: L'étape actuelle du transfert (valeurs possibles:
     ``StepNone``, ``StepSetup``, ``StepPreTasks``, ``StepData``,
     ``StepPostTasks``, ``StepErrorTasks`` ou ``StepFinalization``)
   :resjsonarr number progress: La progression (en octets) du transfert de données
   :resjsonarr number taskNumber: Le numéro du traitement en cours d'exécution
   :resjsonarr string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjsonarr string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)
   :resjsonarr object transferInfo: Des informations de transfert personnalisées sous
     la forme d'une liste de pairs clé:valeur, c'est-à-dire sous forme d'un objet JSON.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/transfers?limit=10&order=desc&rule=1&start=2019-01-01T01:00:00+02:00 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 249

      {
        "transfers": [{
          "id": 1,
          "isServer": false,
          "rule": "règle_1",
          "requester": "toto",
          "requested": "waarp_sftp",
          "protocol": "sftp",
          "localFilepath": "/chemin/local/fichier1",
          "remoteFilepath": "/chemin/distant/fichier1",
          "start": "2019-01-01T02:00:00+02:00",
          "status": "RUNNING",
          "step": "DATA",
          "progress": 123456,
          "transferInfo": { "key1": "val1", "key2": 2, "key3": true }
        },{
          "id": 2,
          "isServer": true,
          "rule": "règle_2",
          "requester": "tata",
          "requested": "sftp_serveur",
          "protocol": "r66",
          "localFilepath": "/chemin/local/fichier2",
          "remoteFilepath": "/chemin/distant/fichier2",
          "start": "2019-01-01T03:00:00+02:00",
          "status": "PLANNED"
        }]
      }
