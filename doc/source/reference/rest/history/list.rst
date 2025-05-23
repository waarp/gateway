Filtrer l'historique
====================

.. http:get:: /api/history

   .. deprecated:: 0.5.0

      Les propriétés ``sourceFilename`` et ``destFilename`` de la réponse ont
      été remplacées par les propriétés ``localFilepath`` et ``remoteFilepath``. 

   Renvoie une liste des entrées de l'historique de transfert remplissant les
   critères donnés en paramètres de requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les transferts seront triés
      Les valeurs possibles sont : ``id+``, ``id-``, ``start+``, ``start-``,
      ``status+``, ``status-``, ``rule+`` et ``rule-``.
      *(défaut: start+)*
   :type sort: string
   :param source: Filtre uniquement les transferts provenant de l'agent renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs sources.
   :type source: string
   :param dest: Filtre uniquement les transferts à destination de l'agent renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs destinations.
   :type dest: string
   :param rule: Filtre uniquement les transferts avec la règle renseignée.
      Peut être renseigné plusieurs fois pour filtrer plusieurs règles.
   :type rule: string
   :param protocol: Filtre uniquement les transferts utilisant le protocole renseigné.
      Peut être renseigné plusieurs fois pour filtrer plusieurs protocoles.
   :type protocol: [sftp]
   :param status: Filtre uniquement les transferts ayant le statut renseigné.
      Valeurs possibles: ``CANCELLED`` ou ``DONE``.
      Peut être renseigné plusieurs fois pour filtrer plusieurs statuts.
   :type status: string
   :param start: Filtre uniquement les transferts ayant commencé après la date
      renseignée. La date doit être renseignée en format ISO 8601 tel qu'il
      est spécifié dans la :rfc:`3339`.
   :type start: date
   :param stop: Filtre uniquement les transferts ayant terminé avant la date
      renseignée. La date doit être renseignée en format ISO 8601 tel qu'il
      est spécifié dans la :rfc:`3339`.
   :type stop: date

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array history: La liste des transferts demandés
   :resjsonarr number id: L'identifiant local du transfert
   :resjsonarr string remoteID: L'identifiant global du transfert
   :resjsonarr bool isServer: Indique si Waarp Gateway est agit en tant que serveur
     (``true``) ou en tant que client (``false``)
   :resjsonarr bool isSend: Indique si le transfert est un envoi (``true``) ou une
     réception (``false``)
   :resjsonarr string requester: Le nom du compte ayant demandé le transfert
   :resjsonarr string requested: Le nom du partenaire avec lequel le transfert a été effectué
   :resjsonarr string protocol: Le protocole utilisé pour le transfert
   :resjsonarr string sourceFilename: *Déprécié*. Le nom du fichier avant le transfert
   :resjsonarr string destFilename: *Déprécié*. Le nom du fichier après le transfert
   :resjsonarr string localFilepath: Le chemin du fichier sur le disque local
   :resjsonarr string remoteFilepath: Le chemin d'accès au fichier sur le partenaire distant
   :resjsonarr number filesize: La taille du fichier (-1 si inconnue)
   :resjsonarr string rule: Le nom de la règle de transfert
   :resjsonarr date start: La date de début du transfert
   :resjsonarr date stop: La date de fin du transfert
   :resjsonarr string status: Le statut final du transfert (``CANCELLED`` ou ``DONE``)
   :resjsonarr string step: La dernière étape du transfert (``NONE``, ``SETUP``, ``PRE TASKS``, ``DATA``, ``POST TASKS``, ``ERROR TASKS`` ou ``FINALIZATION``)
   :resjsonarr number progress: La progression (en octets) du transfert de données
   :resjsonarr number taskNumber: Le numéro du dernier traitement exécuté
   :resjsonarr string errorCode: Le code d'erreur du transfert (si une erreur s'est produite)
   :resjsonarr string errorMsg: Le message d'erreur du transfert (si une erreur s'est produite)
   :resjsonarr object transferInfo: Des informations de transfert personnalisées sous
     la forme d'une liste de pairs clé:valeur, c'est-à-dire sous forme d'un objet JSON.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/history?limit=10&order=desc&rule=regle_sftp&start=2019-01-01T00:00:00+02:00&stop=2019-01-01T04:00:00+02:00 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 293

      {
        "history": [{
          "id": 1,
          "rule": "règle_sftp",
          "source": "compte_sftp_1",
          "dest": "serveur_sftp_1",
          "protocol": "sftp",
          "localPath": "/chemin/local/fichier1",
          "remotePath": "/chemin/distant/fichier1",
          "start": "2019-01-01T01:00:00+02:00",
          "stop": "2019-01-01T02:00:00+02:00",
          "status": "DONE",
          "transferInfo": { "key1": "val1", "key2": 2, "key3": true }
        },{
          "id": 2,
          "rule": "règle_sftp",
          "source": "compte_sftp_2",
          "dest": "serveur_sftp_1",
          "protocol": "sftp",
          "localPath": "/chemin/local/fichier2",
          "remotePath": "/chemin/distant/fichier2",
          "start": "2019-01-01T02:00:00+02:00",
          "stop": "2019-01-01T03:00:00+02:00",
          "status": "CANCELLED",
          "step": "DATA",
          "progress": 123456
        }]
      }
