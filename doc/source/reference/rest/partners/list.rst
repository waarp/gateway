Lister les partenaires
======================

.. http:get:: /api/partners

   Renvoie une liste des partenaires remplissant les critères données en paramètre
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les partenaires seront triés *(défaut: name)*
   :type sortby: [name|address|type]
   :param order: L'ordre dans lequel les partenaires sont triés *(défaut: asc)*
   :type order: [asc|desc]
   :param address: Filtre uniquement les partenaires ayant cette adresse. Peut être renseigné
                   plusieurs fois pour filtrer plusieurs adresses.
   :type address: string
   :param type: Filtre uniquement les partenaires de ce type. Peut être renseigné
                plusieurs fois pour filtrer plusieurs types.
   :type type: [sftp|http]

   **Exemple de requête**

       .. code-block:: http

          GET /api/partners?limit=10&address=waarp.org&address=waarp.fr HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array Partners: La liste des partenaires demandés
   :resjsonarr number ID: L'identifiant unique du partenaire
   :resjsonarr string Name: Le nom du partenaire
   :resjsonarr string Address: L'address (IP ou DNS) du partenaire
   :resjsonarr number Port: Le port sur lequel le partenaire écoute
   :resjsonarr [sftp] Type: Le type de partenaire

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 228

          {
            "Partners": [{
              "ID": 1234,
              "Name": "partenaire1",
              "Addresse": "waarp.fr",
              "Port": 21,
              "Type": "sftp"
            },{
              "ID": 5678,
              "Name": "partenaire2",
              "Addresse": "waarp.org",
              "Port": 8080,
              "Type": "sftp"
            }]
          }